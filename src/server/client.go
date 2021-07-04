package main

import (
	"context"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	con *websocket.Conn
	id  int
}

func (c *Client) sendMessage(msg interface{}) error {
	err := wsjson.Write(context.Background(), c.con, msg)
	if err != nil {
		return err
	}
	return nil
}
