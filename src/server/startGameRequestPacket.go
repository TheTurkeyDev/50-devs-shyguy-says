package server

import (
	"shyguy-says/src/common"
)

func (s *Server) HandleStartGameRequestPacket(packet common.StartGamePacket, c *Client) common.StartGameResponsePacket {
	result := common.StartGameResponsePacket{
		GenericMessage: common.GenericMessage{
			Type: common.StartGameResponse,
		},
		Data: common.StartGameResponseData{
			Valid:   true,
			Message: "",
		},
	}

	room := s.rooms[packet.Data.Room.Name]
	if room.ticking {
		result.Data.Valid = false
		result.Data.Message = "The game is already in progress!"
		return result
	}

	if len(room.clients) < 2 {
		result.Data.Valid = false
		result.Data.Message = "You need atleast 2 players before you can start the game!"
		return result
	}

	go room.runGame()

	room.sendMessageToAllExcluding(result, c)

	return result
}
