package main

import (
	"context"
	"fmt"
	"gowatch/internal/checker"
	"gowatch/internal/handler"
	"gowatch/internal/store"
	"log"
	"net/http"
	"os"
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "GoWatch server is running")
	})

	ctx := context.Background()

	c := checker.New(5, st)
	c.Start(ctx)

	// サーバー起動処理
	log.Println("========== Server starting on :8080 ==========")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
