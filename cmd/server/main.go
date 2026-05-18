package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/oyatulloh/scorepoint/internal/bot"
	"github.com/oyatulloh/scorepoint/internal/hub"
	"github.com/oyatulloh/scorepoint/internal/state"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN muhit o'zgaruvchisi o'rnatilmagan")
	}

	h := hub.New()
	sm := state.New(func(gs state.GameState) {
		h.Broadcast(gs)
	})

	b, err := bot.New(token, sm)
	if err != nil {
		log.Fatalf("bot init xatolik: %v", err)
	}
	go b.Run()

	mux := http.NewServeMux()

	// Scoreboard HTML sahifasi
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// WebSocket — real-time yangilanish
	mux.HandleFunc("/ws", h.HandleWS)

	// REST — boshlang'ich holat olish
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(sm.Get())
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server http://localhost:%s da ishlamoqda", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
