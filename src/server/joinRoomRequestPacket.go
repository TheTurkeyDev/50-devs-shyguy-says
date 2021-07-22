package server

import (
	"shyguy-says/src/common"

	"nhooyr.io/websocket"
)

func (s *Server) HandleJoinRoomRequestPacket(joinMsg common.JoinRoomRequestPacket, ws *websocket.Conn) common.JoinRoomResponsePacket {
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

	if room, exists := s.rooms[joinMsg.Data.Room.Name]; exists {

		if room.ticking {
			result.Data.Valid = false
			result.Data.Message = "You cannot join a room currently in progress!"
			return result
		}

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
			Id:           common.RandUId(8),
			CurrentGuess: -1,
			PlayerNum:    nextPos,
			DisplayName:  joinMsg.Data.DisplayName,
		}

		if joinMsg.Data.Room.Password != room.password {
			result.Data.Valid = false
			result.Data.Message = "Password incorrect!"
			return result
		}

		result.Data.MyId = joinedPlayer.Id
		room.clients = append(room.clients, &Client{
			con:    ws,
			player: joinedPlayer,
		})
		s.clients[ws] = room.roomName

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
			_ = c.sendMessage(joinedMsg)
		}
	} else {
		result.Data.Valid = false
		result.Data.Message = "That room does not exist"
	}

	return result
}
