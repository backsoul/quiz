package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/fasthttp/websocket"
)

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type GameStateMessage struct {
	IsActive  bool   `json:"isActive"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Cliente WebSocket conectado. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()
			log.Printf("Cliente WebSocket desconectado. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error enviando mensaje WebSocket: %v", err)
					delete(h.clients, client)
					client.Close()
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

func (h *Hub) BroadcastGameState(isActive bool, message string) {
	gameState := GameStateMessage{
		IsActive:  isActive,
		Message:   message,
		Timestamp: "2025-07-31T" + "12:00:00Z", // Usar time.Now() en producción
	}

	msg := Message{
		Type: "gameState",
		Data: gameState,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error serializando mensaje: %v", err)
		return
	}

	h.broadcast <- data
}

func (h *Hub) BroadcastMessage(msgType string, data interface{}) {
	msg := Message{
		Type: msgType,
		Data: data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error serializando mensaje: %v", err)
		return
	}

	h.broadcast <- msgData
}
