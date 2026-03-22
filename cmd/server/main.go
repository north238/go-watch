package main

import (
	"fmt"
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

	log.Println("✅ SQLite connected and migrations applied")
	log.Println("✅ GoWatch server initialized successfully")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "GoWatch server is running")
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
