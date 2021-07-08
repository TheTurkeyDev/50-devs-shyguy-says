package main

import (
	"context"
	"shyguy-says/src/common"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	con    *websocket.Conn
	player common.Player
}

func (c *Client) sendMessage(msg interface{}) error {
	err := wsjson.Write(context.Background(), c.con, msg)
	if err != nil {
		return err
		// client.Close(websocket.StatusInternalError, err.Error())
		// delete(clients, client)
	}
	return nil
}
