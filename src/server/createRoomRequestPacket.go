package server

import (
	"shyguy-says/src/common"
	"strings"

	"nhooyr.io/websocket"
)

func (s *Server) HandleCreateRoomRequestPacket(joinMsg common.JoinRoomRequestPacket, ws *websocket.Conn) common.JoinRoomResponsePacket {
	roomName := joinMsg.Data.Room.Name

	_, exists := s.rooms[roomName]
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
		return result
	}

	newId := common.RandUId(8)
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
		return result
	}

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
	s.rooms[roomName] = &room
	s.clients[ws] = roomName

	return result
}
