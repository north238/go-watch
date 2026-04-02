package handler

import (
	"gowatch/internal/websocket"
	"net/http"

	ws "github.com/gorilla/websocket"
)

type WSHandler struct {
	hub *websocket.Hub
}

var upgrader = ws.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 初期化
func NewWSHandler(hub *websocket.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// サーバー起動
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 1. アップグレード
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 2. Client作成 + Hub登録
	client := websocket.NewClient(h.hub, conn)
	h.hub.Register(client)

	// 3. writePump goroutine起動
	go client.WritePump()
}
