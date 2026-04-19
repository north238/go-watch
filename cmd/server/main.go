package main

import (
	"context"
	"fmt"
	"gowatch/internal/checker"
	"gowatch/internal/handler"
	"gowatch/internal/notifier"
	"gowatch/internal/store"
	"gowatch/internal/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	st, err := store.New("data/gowatch.db", "migrations/001_init.sql")
	if err != nil {
		log.Fatalf("failed to initalize store: %v", err)
	}
	defer st.Close()
	log.Println("SQLite connected and migrations applied")

	targetHandler := handler.NewTargetHandler(st)
	http.HandleFunc("POST /api/targets", targetHandler.Create)
	http.HandleFunc("GET /api/targets", targetHandler.Index)
	http.HandleFunc("DELETE /api/targets/{id}", targetHandler.Delete)
	http.HandleFunc("GET /api/targets/{id}/history", targetHandler.History)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "GoWatch server is running")
	})

	ctx, stop := signal.NotifyContext(
		context.Background(), // 親context
		os.Interrupt,         // SIGINT (Ctrl+C)
		syscall.SIGTERM,      // SIGTERM (kill)
	)
	defer stop()

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	var n checker.Notifier
	if webhookURL == "" {
		n = &notifier.NopNotifier{}
	} else {
		n = notifier.NewSlackNotifier(webhookURL)
	}

	// websocket起動
	h := websocket.NewHub()
	go h.Run(ctx)

	// チェッカーの起動
	c := checker.New(5, st, h, n)
	c.Start(ctx)

	wsHandler := handler.NewWSHandler(h)
	http.HandleFunc("GET /ws", wsHandler.ServeWS)

	// サーバー起動処理
	log.Println("========== Server starting on :8080 ==========")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(http.DefaultServeMux),
	}
	go srv.ListenAndServe()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

// CORSミドルウェア
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// プリフライトリクエストへの対応
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
