package websocket

import ws "github.com/gorilla/websocket"

type Client struct {
	hub       *Hub
	websocket *ws.Conn
	send      chan []byte
}

// 初期化
func ClientNew(hub *Hub, conn *ws.Conn) *Client {
	return &Client{
		hub:       hub,
		websocket: conn,
		send:      make(chan []byte, 256),
	}
}

// WebSocketへの書き込み処理
func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.websocket.Close()
	}()

	for message := range c.send {
		if err := c.websocket.WriteMessage(ws.TextMessage, message); err != nil {
			return
		}
	}
}
