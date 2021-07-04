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

var gameState = GameState{}

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
	context.Set("font", "48px serif")

	for i, v := range gameState.Players {
		switch v.CurrentGuess {
		case 0:
			context.Set("fillStyle", "blue")
			context.Call("fillRect", 50+(200*i), 50, 50, 25)
		case 1:
			context.Set("fillStyle", "red")
			context.Call("fillRect", 100+(200*i), 50, 50, 25)
		}
	}

	context.Call("stroke")

	n := rand.Intn(1000)

	if n < 150 && n > 100 {
		n = rand.Intn(3)
		gameState.Players[n+1].CurrentGuess = rand.Intn(2)
	}

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
	group := common.JoinRoomMessage{
		GenericMessage: common.GenericMessage{
			Type: common.JoinRoom,
		},
		Data: common.RoomIdent{
			Name:     getElementValueById("roomName").String(),
			Password: getElementValueById("roomPassword").String(),
		},
	}

	sendQueue <- group
}

func createRoom(this js.Value, inputs []js.Value) {
	roomName := strings.TrimSpace(getElementValueById("roomName").String())
	if len(roomName) == 0 {
		setElementConent("errorText", "Room name cannot be empty!")
		return
	}
	group := common.JoinRoomMessage{
		GenericMessage: common.GenericMessage{
			Type: common.CreateRoom,
		},
		Data: common.RoomIdent{
			Name:     roomName,
			Password: getElementValueById("roomPassword").String(),
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
	for _, p := range gameState.Players {
		if p.Id == gameState.MyId {
			p.CurrentGuess = guess
		}
	}
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
		case common.JoinRoomResult:
		case common.CreateRoomResult:
			var result common.JoinRoomResultMessage
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}

			if !result.Data.Valid {
				setElementConent("errorText", result.Data.Message)
				continue
			}
			gameState.Players[gameState.MyId] = Player{
				Id:           gameState.MyId,
				CurrentGuess: -1,
			}
			initGame()
		}
	}
}

func initGame() {
	getElementById("canvas").Get("style").Set("display", "block")
	getElementById("joinInfo").Get("style").Set("display", "none")
	gameState.SetInRoom(true)
}
