package client

import "shyguy-says/src/common"

func (c *Client) HandleJoinRoomResponsePacket(packet *common.JoinRoomResponsePacket) {
	if !packet.Data.Valid {
		setElementConent("errorText", packet.Data.Message)
		return
	}
	c.MyId = packet.Data.MyId
	c.Room = packet.Data.Room

	for _, p := range packet.Data.Players {
		_, exists := c.Players[p.Id]
		if !exists {
			c.Players[p.Id] = &common.Player{
				Id:           p.Id,
				CurrentGuess: p.CurrentGuess,
				PlayerNum:    p.PlayerNum,
				DisplayName:  p.DisplayName,
			}
			c.PlayersAnimAngles[p.Id] = &PlayerAnimData{
				RedAngle:  0,
				BlueAngle: 0,
			}
		}
	}
	c.initGame()
}
