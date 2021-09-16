package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
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
	width        = 800
	height       = 600
	keyQ         = 113
	keyW         = 119
	armAnimSpeed = 7
)

var sendQueue = make(chan interface{}) // broadcast channel

type Client struct {
	InRoom            bool
	InProgress        bool
	Players           map[string]*common.Player
	PlayersAnimAngles map[string]*PlayerAnimData
	MyId              string
	Room              common.RoomIdent
	ShyGuyDisplay     int
	ShyGuyAngles      *PlayerAnimData
	Round             int
	RoundStatus       int
	TitleMessages     []*common.TitleMessageData
}

func NewInstance() *Client {
	client := &Client{
		InRoom:            false,
		InProgress:        false,
		Players:           make(map[string]*common.Player),
		PlayersAnimAngles: make(map[string]*PlayerAnimData),
		MyId:              "",
		Room: common.RoomIdent{
			Name:     "",
			Password: "",
		},
		ShyGuyDisplay: -1,
		ShyGuyAngles: &PlayerAnimData{
			RedAngle:  0,
			BlueAngle: 0,
		},
		Round:         0,
		RoundStatus:   -1,
		TitleMessages: []*common.TitleMessageData{},
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
	go c.tickLoop()
	c.initWebSocket()

	<-done
}

func (c *Client) initWebSocket() {
	fmt.Println("Starting Websocket!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ws, _, err := websocket.Dial(ctx, "https://test.theturkey.dev/ws", nil)
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

func (c *Client) tick() {
	for _, tm := range c.TitleMessages {
		tm.DisplayTime -= 1
	}

	for id, p := range c.Players {
		angles := c.PlayersAnimAngles[id]
		updateAngles(p.CurrentGuess, angles)
	}
	updateAngles(c.ShyGuyDisplay, c.ShyGuyAngles)
}

func updateAngles(currentGuess int, angles *PlayerAnimData) {
	if currentGuess == 0 && angles.BlueAngle < 85 {
		angles.BlueAngle += armAnimSpeed
	} else if currentGuess != 0 && angles.BlueAngle > 0 {
		angles.BlueAngle -= armAnimSpeed
	}

	if currentGuess == 1 && angles.RedAngle < 85 {
		angles.RedAngle += armAnimSpeed
	} else if currentGuess != 1 && angles.RedAngle > 0 {
		angles.RedAngle -= armAnimSpeed
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
	context.Set("textAlign", "center")
	context.Set("font", "24px serif")

	for _, v := range c.Players {
		context.Set("fillStyle", "black")
		context.Call("fillText", v.DisplayName, 103+(200*v.PlayerNum), 30)
		context.Call("drawImage", getElementById("penguin_body"), 70+(200*v.PlayerNum), 40, 75, 129)

		prX := 127 + (200 * v.PlayerNum)
		pbX := 18 + (200 * v.PlayerNum)
		pY := 92
		pW := 69
		hY := 61
		offset := 5

		angles := c.PlayersAnimAngles[v.Id]

		// Red flag flipper
		context.Call("save")
		context.Call("translate", prX+offset, pY)
		context.Call("rotate", -(angles.RedAngle)*math.Pi/180)
		context.Call("translate", -(prX + offset), -pY)
		context.Call("drawImage", getElementById("penguin_flipper_red_flag"), prX, pY, pW, hY)
		context.Call("restore")

		// Blue flag flipper
		context.Call("save")
		context.Call("translate", pbX+(pW-offset), pY)
		context.Call("rotate", (angles.BlueAngle)*math.Pi/180)
		context.Call("translate", -pbX-(pW-offset), -pY)
		context.Call("drawImage", getElementById("penguin_flipper_blue_flag"), pbX, pY, pW, hY)
		context.Call("restore")
	}

	context.Set("fillStyle", "black")
	context.Call("fillText", "Pengu", 400, 590)
	context.Call("drawImage", getElementById("penguin_behind"), 335, 300)

	prX := 450
	pbX := 250
	pY := 400
	pW := 118
	hY := 122
	offset := 5
	// Red flag flipper
	context.Call("save")
	context.Call("translate", prX+offset, pY)
	context.Call("rotate", -(c.ShyGuyAngles.RedAngle)*math.Pi/180)
	context.Call("translate", -(prX + offset), -pY)
	context.Call("drawImage", getElementById("penguin_flipper_red_flag"), prX, pY, pW, hY)
	context.Call("restore")

	// Blue flag flipper
	context.Call("save")
	context.Call("translate", pbX+(pW-offset), pY)
	context.Call("rotate", (c.ShyGuyAngles.BlueAngle)*math.Pi/180)
	context.Call("translate", -pbX-(pW-offset), -pY)
	context.Call("drawImage", getElementById("penguin_flipper_blue_flag"), pbX, pY, pW, hY)
	context.Call("restore")

	if c.InProgress {
		context.Set("fillStyle", "black")
		context.Call("fillText", fmt.Sprintf("Round: %d", c.Round), 50, 550)
		context.Call("fillText", fmt.Sprintf("Round Status: %d", c.RoundStatus), 50, 575)
	}

	context.Set("textAlign", "center")
	context.Set("textBaseline", "middle")
	for i := len(c.TitleMessages) - 1; i >= 0; i-- {
		tm := c.TitleMessages[i]
		if tm.DisplayTime == 0 {
			c.TitleMessages = append(c.TitleMessages[:i], c.TitleMessages[i+1:]...)
		} else {
			context.Set("fillStyle", tm.Color)
			switch tm.Location {
			case 0:
				context.Set("font", "128px serif")
				context.Call("fillText", tm.Value, width/2, height/2)
			case 1:
				context.Set("font", "96px serif")
				context.Call("fillText", tm.Value, width/2, (height/2)+75)
			}
		}
	}

	context.Set("font", "16px serif")
	context.Set("textAlign", "start")
	context.Set("textAlign", "top")

	context.Call("stroke")

	// for i := 0; i < 50; i++ {
	// 	context.Call("moveTo", getRandomNum()*width, getRandomNum()*height)
	// 	context.Call("lineTo", getRandomNum()*width, getRandomNum()*height)
	// }
}

func (c *Client) tickLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / 20)) // 20 tps
	defer ticker.Stop()                                      // Not gonna happen, but good practice becasue thats the only way these get GC'd
	for range ticker.C {
		c.tick()
	}
}

func (c *Client) frameLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / 60)) // 30 fps
	defer ticker.Stop()                                      // Not gonna happen, but good practice becasue thats the only way these get GC'd
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
			c.PlayersAnimAngles[real.Data.Player.Id] = &PlayerAnimData{
				BlueAngle: 0,
				RedAngle:  0,
			}
			c.Players[real.Data.Player.Id] = &real.Data.Player
		case *common.UserInputPacket:
			for id, p := range c.Players {
				if id == real.Data.PlayerId {
					p.CurrentGuess = real.Data.Input
				}
			}
		case *common.UserLeaveRoomPacket:
			delete(c.Players, real.Data.PlayerId)
			delete(c.PlayersAnimAngles, real.Data.PlayerId)
		case *common.StartGameResponsePacket:
			c.HandleStartGameResponsePacket(real)
		case *common.GameOverPacket:
			c.InProgress = false
			clearErrorMsg()
			getElementById("gameInputFields").Get("style").Set("display", "block")
		case *common.ShyGuyDisplayPacket:
			c.ShyGuyDisplay = real.Data.Input
		case *common.RoundUpdatePacket:
			c.Round = real.Data.Round
			c.RoundStatus = real.Data.RoundStatus
		case *common.TitleMessagePacket:
			c.TitleMessages = append(c.TitleMessages, &real.Data)
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
	case common.RoundUpdate:
		target = &common.RoundUpdatePacket{}
	case common.TitleMessage:
		target = &common.TitleMessagePacket{}
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
