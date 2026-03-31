package handler

import (
	"gowatch/internal/websocket"
	"net/http"
)

type WSHandler struct {
	hub *websocket.Hub
}

// 初期化
func NewWSHundler(hub *websocket.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 1. アップグレード

	// 2. Client作成 + Hub登録

	// 3. writePump goroutine起動
}
