package client

import "shyguy-says/src/common"

func (c *Client) HandleStartGameResponsePacket(packet *common.StartGameResponsePacket) {
	if !packet.Data.Valid {
		setElementConent("errorText", packet.Data.Message)
		return
	}

	clearErrorMsg()
	getElementById("gameInputFields").Get("style").Set("display", "none")
}
