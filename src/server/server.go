package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shyguy-says/src/common"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Server struct {
	clients map[*websocket.Conn]string
	rooms   map[string]*Room
}

func NewInstance() *Server {
	server := &Server{
		clients: make(map[*websocket.Conn]string), // connected clients
		rooms:   make(map[string]*Room),
	}

	server.run()

	return server
}

func (s *Server) run() {
	log.SetFlags(0)

	fs := http.FileServer(http.Dir("./dist"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", s.handleConnections)

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
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
	s.clients[ws] = ""

	for {
		// Read in a new message as JSON and map it to a Message object
		_, bytes, err := ws.Read(r.Context())
		if err != nil {
			log.Printf("error: %v", err)
			s.onDisconnect(ws)
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
				continue
			}

			result := s.HandleJoinRoomRequestPacket(joinMsg, ws)
			s.sendMessage(ws, result)

		case common.CreateRoomRequest:
			var joinMsg common.JoinRoomRequestPacket
			if err := json.Unmarshal(bytes, &joinMsg); err != nil {
				fmt.Println("error:", err)
				continue
			}

			result := s.HandleCreateRoomRequestPacket(joinMsg, ws)
			s.sendMessage(ws, result)
		case common.UserInput:
			var result common.UserInputPacket
			if err := json.Unmarshal(bytes, &result); err != nil {
				fmt.Println("error:", err)
				continue
			}

			roomName := s.clients[ws]
			room := s.rooms[roomName]
			client := s.getClientForWS(ws, roomName)

			if room.onUserInputChange(client, result.Data.Input) {
				room.sendMessageToAll(result)
			}
			// c.player.CurrentGuess = result.Data.Input
		case common.StartGame:
			var packet common.StartGamePacket
			if err := json.Unmarshal(bytes, &packet); err != nil {
				fmt.Println("error:", err)
				continue
			}

			roomName := s.clients[ws]
			result := s.HandleStartGameRequestPacket(packet, s.getClientForWS(ws, roomName))
			s.sendMessage(ws, result)
		}

	}
}

func (s *Server) onDisconnect(ws *websocket.Conn) {
	roomName := s.clients[ws]
	room := s.rooms[roomName]

	leavingClient := s.getClientForWS(ws, roomName)

	delete(s.clients, ws)

	room.onClientLeave(leavingClient)

	if len(room.Clients) == 0 {
		room.stopGame()
		delete(s.rooms, roomName)
		return
	}

	room.sendMessageToAll(common.UserLeaveRoomPacket{
		GenericMessage: common.GenericMessage{
			Type: common.UserLeave,
		},
		Data: common.UserLeaveRoomData{
			PlayerId: leavingClient.player.Id,
			Reason:   "Lost Connection",
		},
	})
}

func (s *Server) sendMessage(client *websocket.Conn, msg interface{}) {
	err := wsjson.Write(context.Background(), client, msg)
	if err != nil {
		log.Printf("error: %v", err)
		client.Close(websocket.StatusInternalError, err.Error())
		delete(s.clients, client)
	}
}

func (s *Server) getClientForWS(ws *websocket.Conn, roomName string) *Client {
	for _, c := range s.rooms[roomName].Clients {
		if c.con == ws {
			return c
		}
	}
	return nil
}
