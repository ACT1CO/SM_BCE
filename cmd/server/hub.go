package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients     map[*Client]bool
	history     []Message
	historyPath string
	mu          sync.Mutex
}

func NewHub(historyPath string) *Hub {
	hub := &Hub{clients: make(map[*Client]bool), historyPath: historyPath}
	hub.loadHistory()
	return hub
}

func (h *Hub) loadHistory() {
	data, err := os.ReadFile(h.historyPath)
	if err != nil {
		return
	}
	var file HistoryFile
	if err := json.Unmarshal(data, &file); err != nil {
		log.Println("history parse error:", err)
		return
	}
	h.history = file.Messages
}

func (h *Hub) saveHistoryLocked() {
	if len(h.history) > 500 {
		h.history = h.history[len(h.history)-500:]
	}
	data, err := json.MarshalIndent(HistoryFile{Messages: h.history}, "", "  ")
	if err != nil {
		log.Println("history marshal error:", err)
		return
	}
	if err := os.WriteFile(h.historyPath, data, 0600); err != nil {
		log.Println("history save error:", err)
	}
}

func (h *Hub) Add(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	h.Send(client, Message{Type: "hello", ClientID: client.User.ID, User: client.User})
	h.SendHistory(client)
	h.Broadcast(Message{Type: "system", Text: client.User.Name + " присоединился к чату", Time: now()})
	h.BroadcastUsers()
}

func (h *Hub) Remove(client *Client) {
	removed := false
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		removed = true
	}
	h.mu.Unlock()
	if !removed {
		return
	}
	h.Broadcast(Message{Type: "system", Text: client.User.Name + " вышел из чата", Time: now()})
	h.BroadcastUsers()
}

func (h *Hub) AddHistory(msg Message) {
	h.mu.Lock()
	h.history = append(h.history, msg)
	h.saveHistoryLocked()
	h.mu.Unlock()
}

func (h *Hub) SendHistory(client *Client) {
	h.mu.Lock()
	history := make([]Message, 0, len(h.history))
	for _, msg := range h.history {
		if !msg.Private || msg.From == client.User.ID || msg.To == client.User.ID {
			history = append(history, msg)
		}
	}
	h.mu.Unlock()
	h.Send(client, Message{Type: "history", Messages: history})
}

func (h *Hub) Send(client *Client, msg Message) {
	client.writeMu.Lock()
	defer client.writeMu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("json marshal error:", err)
		return
	}
	if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Println("write message error:", err)
		client.Conn.Close()
		h.mu.Lock()
		delete(h.clients, client)
		h.mu.Unlock()
	}
}

func (h *Hub) Broadcast(msg Message) {
	for _, client := range h.AllClients() {
		h.Send(client, msg)
	}
}

func (h *Hub) BroadcastUsers() {
	h.mu.Lock()
	seen := make(map[string]bool)
	users := make([]User, 0, len(h.clients))
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
		if seen[client.User.ID] {
			continue
		}
		seen[client.User.ID] = true
		users = append(users, client.User)
	}
	h.mu.Unlock()
	msg := Message{Type: "users", Users: users}
	for _, client := range clients {
		h.Send(client, msg)
	}
}

func (h *Hub) AllClients() []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

func (h *Hub) ClientsByID(id string) []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := make([]*Client, 0)
	for client := range h.clients {
		if client.User.ID == id {
			clients = append(clients, client)
		}
	}
	return clients
}
