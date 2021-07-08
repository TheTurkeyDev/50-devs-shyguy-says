package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"shyguy-says/src/common"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	width  = 800
	height = 600
	keyQ   = 113
	keyW   = 119
)

var gameState = GameState{
	InRoom:  false,
	Players: make(map[string]*common.Player),
	MyId:    "",
	Room: common.RoomIdent{
		Name:     "",
		Password: "",
	},
}

var sendQueue = make(chan interface{}) // broadcast channel

func render() {
	if !gameState.InRoom {
		return
	}

	var canvas js.Value = getElementById("canvas")

	var context js.Value = canvas.Call("getContext", "2d")

	// reset
	canvas.Set("height", height)
	canvas.Set("width", width)
	context.Call("clearRect", 0, 0, width, height)

	context.Call("beginPath")
	context.Set("font", "16px serif")

	for _, v := range gameState.Players {
		fmt.Println(v)
		context.Set("fillStyle", "black")
		context.Call("fillText", v.DisplayName, 50+(200*v.PlayerNum), 20)
		switch v.CurrentGuess {
		case 0:
			context.Set("fillStyle", "blue")
			context.Call("fillRect", 50+(200*v.PlayerNum), 50, 50, 25)
		case 1:
			context.Set("fillStyle", "red")
			context.Call("fillRect", 100+(200*v.PlayerNum), 50, 50, 25)
		}
	}

	context.Call("stroke")

	// for i := 0; i < 50; i++ {
	// 	context.Call("moveTo", getRandomNum()*width, getRandomNum()*height)
	// 	context.Call("lineTo", getRandomNum()*width, getRandomNum()*height)
	// }
}

func frameLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / 60))
	defer ticker.Stop() // Not gonna happen, but good practice becasue thats the only way these get GC'd
	for range ticker.C {
		render()
	}
}

func joinRoom(this js.Value, inputs []js.Value) {
	displayName := getElementValueById("displayName").String()

	if displayName == "" {
		setElementConent("errorText", "Display Name cannot be empty!")
		return
	}

	group := common.JoinRoomRequestPacket{
		GenericMessage: common.GenericMessage{
			Type: common.JoinRoomRequest,
		},
		Data: common.JoinRoomRequestData{
			Room: common.RoomIdent{
				Name:     getElementValueById("roomName").String(),
				Password: getElementValueById("roomPassword").String(),
			},
			DisplayName: displayName,
		},
	}

	sendQueue <- group
}

func createRoom(this js.Value, inputs []js.Value) {
	displayName := getElementValueById("displayName").String()

	if displayName == "" {
		setElementConent("errorText", "Display Name cannot be empty!")
		return
	}

	roomName := strings.TrimSpace(getElementValueById("roomName").String())
	if len(roomName) == 0 {
		setElementConent("errorText", "Room name cannot be empty!")
		return
	}
	group := common.JoinRoomRequestPacket{
		GenericMessage: common.GenericMessage{
			Type: common.CreateRoomRequest,
		},
		Data: common.JoinRoomRequestData{
			Room: common.RoomIdent{
				Name:     roomName,
				Password: getElementValueById("roomPassword").String(),
			},
			DisplayName: displayName,
		},
	}

	sendQueue <- group
}

func onClick(this js.Value, inputs []js.Value) {
	println("click")
}

func keyPress(this js.Value, inputs []js.Value) interface{} {
	if inputs[0].Get("keyCode").Int() == keyQ {
		setCurrentGuess(0)
	} else if inputs[0].Get("keyCode").Int() == keyW {
		setCurrentGuess(1)
	}
	return 1
}

func setCurrentGuess(guess int) {
	packet := common.UserInputPacket{
		GenericMessage: common.GenericMessage{
			Type: common.UserInput,
		},
		Data: common.UserInputData{
			PlayerId: gameState.MyId,
			Input:    guess,
		},
	}
	sendQueue <- packet
}

func main() {
	fmt.Println("Web Assembly Running!")
	rand.New(rand.NewSource(time.Now().UnixNano()))
	// see https://tip.golang.org/pkg/syscall/js/?GOOS=js&GOARCH=wasm#NewCallback
	done := make(chan struct{})
	bindEventListener("canvas", onClick)

	js.
		Global().
		Get("document").
		Set("onkeypress", js.FuncOf(keyPress))

	bindEventListener("joinBtn", joinRoom)
	bindEventListener("createRoomBtn", createRoom)

	go frameLoop()
	initWebSocket()

	<-done
}

func initWebSocket() {
	fmt.Println("Starting Websocket!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "http://localhost:8000/ws", nil)
	if err != nil {
		print("Error 1!")
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	hold := &sync.WaitGroup{}

	hold.Add(2)

	go sendMessages(c, hold)
	go readMessages(c, hold)

	hold.Wait()
}

func sendMessages(server *websocket.Conn, hold *sync.WaitGroup) {
	defer hold.Done()
	for msg := range sendQueue {
		// Send it out to every client that is currently connected
		err := wsjson.Write(context.Background(), server, msg)
		if err != nil {
			log.Printf("error: %v", err)
			server.Close(websocket.StatusInternalError, err.Error())
			return
		}
	}
}

func readMessages(server *websocket.Conn, hold *sync.WaitGroup) {
	defer hold.Done()
	for {
		// Read in a new message as JSON and map it to a Message object
		_, bytes, err := server.Read(context.Background())
		if err != nil {
			log.Printf("error: %v", err)
			break
		}

		var message common.GenericMessage
		if err := json.Unmarshal(bytes, &message); err != nil {
			fmt.Println("error:", err)
		}

		switch message.Type {
		case common.JoinRoomResponse:
			var result common.JoinRoomResponsePacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}

			if !result.Data.Valid {
				setElementConent("errorText", result.Data.Message)
				continue
			}
			gameState.MyId = result.Data.MyId
			gameState.Room = result.Data.Room

			for _, p := range result.Data.Players {
				_, exists := gameState.Players[p.Id]
				if !exists {
					gameState.Players[p.Id] = &common.Player{
						Id:           p.Id,
						CurrentGuess: p.CurrentGuess,
						PlayerNum:    p.PlayerNum,
						DisplayName:  p.DisplayName,
					}
				}
			}
			initGame()
		case common.CreateRoomResponse:
			var result common.JoinRoomResponsePacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}

			if !result.Data.Valid {
				setElementConent("errorText", result.Data.Message)
				continue
			}
			gameState.MyId = result.Data.MyId
			gameState.Room = result.Data.Room

			for _, p := range result.Data.Players {
				gameState.Players[p.Id] = &common.Player{
					Id:           p.Id,
					CurrentGuess: p.CurrentGuess,
					PlayerNum:    p.PlayerNum,
					DisplayName:  p.DisplayName,
				}
			}

			initGame()
		case common.UserJoin:
			var result common.UserJoinRoomPacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}
			gameState.Players[result.Data.Player.Id] = &result.Data.Player
		case common.UserInput:
			var result common.UserInputPacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}
			for id, p := range gameState.Players {
				if id == result.Data.PlayerId {
					p.CurrentGuess = result.Data.Input
				}
			}
		case common.UserLeave:
			var result common.UserInputPacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}

			delete(gameState.Players, result.Data.PlayerId)
		}
	}
}

func initGame() {
	getElementById("canvas").Get("style").Set("display", "block")
	getElementById("joinInfo").Get("style").Set("display", "none")
	gameState.SetInRoom(true)
}
