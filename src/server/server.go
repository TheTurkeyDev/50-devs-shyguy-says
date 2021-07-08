package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"shyguy-says/src/common"
	"strings"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	clients = make(map[*websocket.Conn]string) // connected clients
	rooms   = make(map[string]*Room)
)

func main() {
	log.SetFlags(0)

	fs := http.FileServer(http.Dir("./dist"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Print("Connection!")

	// Make sure we close the connection when the function returns
	defer ws.Close(websocket.StatusInternalError, "Closing!")

	// Register our new client
	clients[ws] = ""

	for {
		// Read in a new message as JSON and map it to a Message object
		_, bytes, err := ws.Read(r.Context())
		if err != nil {
			log.Printf("error: %v", err)
			onDisconnect(ws)
			break
		}

		var message common.GenericMessage
		if err := json.Unmarshal(bytes, &message); err != nil {
			fmt.Println("error:", err)
		}

		switch message.Type {
		case common.JoinRoomRequest:
			var joinMsg common.JoinRoomRequestPacket
			if err := json.Unmarshal(bytes, &joinMsg); err != nil {
				fmt.Println("error:", err)
			}

			result := common.JoinRoomResponsePacket{
				GenericMessage: common.GenericMessage{
					Type: common.JoinRoomResponse,
				},
				Data: common.JoinRoomResponseData{
					Room:    joinMsg.Data.Room,
					Valid:   true,
					Message: "",
					MyId:    "",
					Players: []common.Player{},
				},
			}

			room, exists := rooms[joinMsg.Data.Room.Name]
			if exists {
				nextPos := 0
				contained := true
				for contained {
					contained = false
					for _, c := range room.clients {
						if c.player.PlayerNum == nextPos {
							nextPos += 1
							contained = true
							break
						}
					}
				}

				joinedPlayer := common.Player{
					Id:           randUId(8),
					CurrentGuess: -1,
					PlayerNum:    nextPos,
					DisplayName:  joinMsg.Data.DisplayName,
				}

				if joinMsg.Data.Room.Password != room.password {
					result.Data.Valid = false
					result.Data.Message = "Password incorrect!"

				} else {
					result.Data.MyId = joinedPlayer.Id
					room.clients = append(room.clients, &Client{
						con:    ws,
						player: joinedPlayer,
					})
					clients[ws] = room.roomName

					for _, c := range room.clients {
						result.Data.Players = append(result.Data.Players, c.player)
						joinedMsg := common.UserJoinRoomPacket{
							GenericMessage: common.GenericMessage{
								Type: common.UserJoin,
							},
							Data: common.UserJoinRoomData{
								Room:   joinMsg.Data.Room,
								Player: joinedPlayer,
							},
						}
						c.sendMessage(joinedMsg)
					}
				}
			} else {
				result.Data.Valid = false
				result.Data.Message = "That room does not exist"
			}

			sendMessage(ws, result)

		case common.CreateRoomRequest:
			var joinMsg common.JoinRoomRequestPacket
			if err := json.Unmarshal(bytes, &joinMsg); err != nil {
				fmt.Println("error:", err)
			}

			roomName := joinMsg.Data.Room.Name

			_, exists := rooms[roomName]
			if exists {
				result := common.JoinRoomResponsePacket{
					GenericMessage: common.GenericMessage{
						Type: common.CreateRoomResponse,
					},
					Data: common.JoinRoomResponseData{
						Room:    joinMsg.Data.Room,
						Valid:   false,
						Message: "A room with that name already exists!",
						MyId:    "",
						Players: []common.Player{},
					},
				}
				sendMessage(ws, result)
				continue
			}

			newId := randUId(8)
			roomPlayers := []common.Player{
				{
					Id:           newId,
					PlayerNum:    0,
					CurrentGuess: -1,
					DisplayName:  joinMsg.Data.DisplayName,
				},
			}

			result := common.JoinRoomResponsePacket{
				GenericMessage: common.GenericMessage{
					Type: common.CreateRoomResponse,
				},
				Data: common.JoinRoomResponseData{
					Room:    joinMsg.Data.Room,
					Valid:   true,
					Message: "",
					MyId:    newId,
					Players: roomPlayers,
				},
			}

			if len(strings.TrimSpace(roomName)) == 0 {
				result.Data.Valid = false
				result.Data.Message = "Room name cannot be empty!"
			} else {
				room := Room{
					roomName: roomName,
					password: joinMsg.Data.Room.Password,
					clients: []*Client{{
						con: ws,
						player: common.Player{
							Id:           result.Data.MyId,
							CurrentGuess: -1,
							PlayerNum:    roomPlayers[0].PlayerNum,
							DisplayName:  roomPlayers[0].DisplayName,
						},
					}},
				}
				rooms[roomName] = &room
				clients[ws] = roomName
			}

			sendMessage(ws, result)
		case common.UserInput:
			var result common.UserInputPacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
			}

			roomName := clients[ws]

			// c.player.CurrentGuess = result.Data.Input
			for _, c := range rooms[roomName].clients {
				c.sendMessage(result)
			}
		}
	}
}

func onDisconnect(ws *websocket.Conn) {
	roomName := clients[ws]

	leavingClient := &Client{}
	for _, c := range rooms[roomName].clients {
		if c.con == ws {
			leavingClient = c
			break
		}
	}

	delete(clients, ws)
	rc := rooms[roomName].clients
	for i, c := range rc {
		if c == leavingClient {
			rc = append(rc[:i], rc[i+1:]...)
			break
		}
	}
	rooms[roomName].clients = rc

	if len(rooms[roomName].clients) == 0 {
		delete(rooms, roomName)
		return
	}

	for _, c := range rooms[roomName].clients {
		c.sendMessage(common.UserLeaveRoomPacket{
			GenericMessage: common.GenericMessage{
				Type: common.UserLeave,
			},
			Data: common.UserLeaveRoomData{
				PlayerId: leavingClient.player.Id,
				Reason:   "Lost Connection",
			},
		})
	}
}

func sendMessage(client *websocket.Conn, msg interface{}) {
	err := wsjson.Write(context.Background(), client, msg)
	if err != nil {
		log.Printf("error: %v", err)
		client.Close(websocket.StatusInternalError, err.Error())
		delete(clients, client)
	}
}

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randUId(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
