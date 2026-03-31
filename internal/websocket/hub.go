package websocket

import "context"

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

// 初期化
func NewHub() *Hub {
	clients := make(map[*Client]bool)

	register := make(chan *Client)

	unregister := make(chan *Client)

	broadcast := make(chan []byte)

	return &Hub{
		clients:    clients,
		register:   register,
		unregister: unregister,
		broadcast:  broadcast,
	}
}

// メイン
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register: // 新規接続
			h.clients[client] = true
		case client := <-h.unregister: // 切断
			delete(h.clients, client)
		case message := <-h.broadcast: // 全員に送信
			for client := range h.clients {
				client.send <- message
			}
		}
	}
}

// registerを送る
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// messageを代入
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}
