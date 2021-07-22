package client

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

var sendQueue = make(chan interface{}) // broadcast channel

type Client struct {
	InRoom        bool
	Players       map[string]*common.Player
	MyId          string
	Room          common.RoomIdent
	ShyGuyDisplay int
}

func NewInstance() *Client {
	client := &Client{
		InRoom:  false,
		Players: make(map[string]*common.Player),
		MyId:    "",
		Room: common.RoomIdent{
			Name:     "",
			Password: "",
		},
		ShyGuyDisplay: -1,
	}

	client.run()

	return client
}

func (c *Client) run() {
	fmt.Println("Web Assembly Running!")
	rand.New(rand.NewSource(time.Now().UnixNano()))
	// see https://tip.golang.org/pkg/syscall/js/?GOOS=js&GOARCH=wasm#NewCallback
	done := make(chan struct{})

	c.initJSBindings()

	go c.frameLoop()
	c.initWebSocket()

	<-done
}

func (c *Client) initWebSocket() {
	fmt.Println("Starting Websocket!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ws, _, err := websocket.Dial(ctx, "http://localhost:8000/ws", nil)
	if err != nil {
		print("Error 1!")
		return
	}
	defer ws.Close(websocket.StatusInternalError, "the sky is falling")

	hold := &sync.WaitGroup{}

	hold.Add(2)

	go c.sendMessages(ws, hold)
	go c.readMessages(ws, hold)

	hold.Wait()
}

func (c *Client) sendMessages(server *websocket.Conn, hold *sync.WaitGroup) {
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

func (c *Client) render() {
	if !c.InRoom {
		return
	}

	canvas := getElementById("canvas")

	context := canvas.Call("getContext", "2d")

	// reset
	canvas.Set("height", height)
	canvas.Set("width", width)
	context.Call("clearRect", 0, 0, width, height)

	context.Call("beginPath")
	context.Set("font", "16px serif")

	for _, v := range c.Players {
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

	context.Set("fillStyle", "black")
	context.Call("fillText", "ShyGuy", 500, 500)
	switch c.ShyGuyDisplay {
	case 0:
		context.Set("fillStyle", "blue")
		context.Call("fillRect", 500, 500, 50, 25)
	case 1:
		context.Set("fillStyle", "red")
		context.Call("fillRect", 550, 500, 50, 25)
	}

	context.Call("stroke")

	// for i := 0; i < 50; i++ {
	// 	context.Call("moveTo", getRandomNum()*width, getRandomNum()*height)
	// 	context.Call("lineTo", getRandomNum()*width, getRandomNum()*height)
	// }
}

func (c *Client) frameLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / 60))
	defer ticker.Stop() // Not gonna happen, but good practice becasue thats the only way these get GC'd
	for range ticker.C {
		c.render()
	}
}

func (c *Client) readMessages(server *websocket.Conn, hold *sync.WaitGroup) {
	defer hold.Done()
	for {
		// Read in a new message as JSON and map it to a Message object
		_, bytes, err := server.Read(context.Background())
		if err != nil {
			log.Printf("error: %v", err)
			break
		}

		message := getMessage(bytes)

		switch real := message.(type) {
		case *common.JoinRoomResponsePacket:
			if real.Type == common.JoinRoomRequest {
				c.HandleJoinRoomResponsePacket(real)
			} else {
				c.HandleCreateRoomResponsePacket(real)
			}
		case *common.UserJoinRoomPacket:
			c.Players[real.Data.Player.Id] = &real.Data.Player
		case *common.UserInputPacket:
			for id, p := range c.Players {
				if id == real.Data.PlayerId {
					p.CurrentGuess = real.Data.Input
				}
			}
		case *common.UserLeaveRoomPacket:
			delete(c.Players, real.Data.PlayerId)
		case *common.StartGameResponsePacket:
			c.HandleStartGameResponsePacket(real)
		case *common.GameOverPacket:
			clearErrorMsg()
			getElementById("gameInputFields").Get("style").Set("display", "block")
		case *common.ShyGuyDisplayPacket:
			c.ShyGuyDisplay = real.Data.Input
		}
	}
}

func getMessage(bytes []byte) interface{} {
	var message common.GenericMessage
	if err := json.Unmarshal(bytes, &message); err != nil {
		fmt.Println("error:", err)
		return nil
	}

	var target interface{}

	switch message.Type {
	case common.JoinRoomResponse:
		target = &common.JoinRoomResponsePacket{}
	case common.CreateRoomResponse:
		target = &common.JoinRoomResponsePacket{}
	case common.UserJoin:
		target = &common.UserJoinRoomPacket{}
	case common.UserInput:
		target = &common.UserInputPacket{}
	case common.UserLeave:
		target = &common.UserLeaveRoomPacket{}
	case common.StartGameResponse:
		target = &common.StartGameResponsePacket{}
	case common.GameOver:
		target = &common.GameOverPacket{}
	case common.ShyGuyDsiplay:
		target = &common.ShyGuyDisplayPacket{}
	}

	if err := json.Unmarshal(bytes, &target); err != nil {
		fmt.Println("error:", err)
		return nil
	}
	return target
}

func (c *Client) initGame() {
	getElementById("canvas").Get("style").Set("display", "block")
	getElementById("joinInfo").Get("style").Set("display", "none")
	getElementById("gameInputFields").Get("style").Set("display", "block")
	clearErrorMsg()
	c.InRoom = true
}

func (c *Client) joinRoom(_ js.Value, _ []js.Value) {
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

func (c *Client) createRoom(_ js.Value, _ []js.Value) {
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

func (c *Client) startgame(_ js.Value, _ []js.Value) {
	group := common.StartGamePacket{
		GenericMessage: common.GenericMessage{
			Type: common.StartGame,
		},
		Data: common.StartGameData{
			Room: c.Room,
		},
	}

	sendQueue <- group
}

func (c *Client) onClick(_ js.Value, _ []js.Value) {
	println("click")
}

func (c *Client) keyPress(_ js.Value, inputs []js.Value) interface{} {
	if inputs[0].Get("keyCode").Int() == keyQ {
		c.setCurrentGuess(0)
	} else if inputs[0].Get("keyCode").Int() == keyW {
		c.setCurrentGuess(1)
	}
	return 1
}

func (c *Client) setCurrentGuess(guess int) {
	packet := common.UserInputPacket{
		GenericMessage: common.GenericMessage{
			Type: common.UserInput,
		},
		Data: common.UserInputData{
			PlayerId: c.MyId,
			Input:    guess,
		},
	}
	sendQueue <- packet
}
