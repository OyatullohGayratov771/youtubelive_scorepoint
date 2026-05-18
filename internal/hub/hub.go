package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients sync.Map
	send    chan []byte
}

func New() *Hub {
	h := &Hub{send: make(chan []byte, 64)}
	go h.run()
	return h
}

func (h *Hub) run() {
	for data := range h.send {
		h.clients.Range(func(k, _ any) bool {
			conn := k.(*websocket.Conn)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				conn.Close()
				h.clients.Delete(conn)
			}
			return true
		})
	}
}

func (h *Hub) Broadcast(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("hub: marshal error: %v", err)
		return
	}
	select {
	case h.send <- data:
	default:
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("hub: ws upgrade: %v", err)
		return
	}
	h.clients.Store(conn, true)
	go func() {
		defer func() {
			h.clients.Delete(conn)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
