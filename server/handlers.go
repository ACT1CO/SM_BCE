package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: func(r *http.Request) bool { return true }}

func keyHandler(keys *KeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, key, allKeys := keys.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{"day": day, "key": key, "keys": allKeys})
	}
}

func registerHandler(users *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		user, err := users.Register(req.Name, req.Tag)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]User{"user": user})
	}
}

func wsHandler(hub *Hub, users *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		user, ok := users.ByID(id)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unknown user"})
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("websocket upgrade error:", err)
			return
		}
		client := &Client{User: user, Conn: conn}
		hub.Add(client)
		defer func() { conn.Close(); hub.Remove(client) }()
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				log.Println("read message error:", err)
				break
			}
			handleIncoming(hub, users, client, raw)
		}
	}
}

func handleIncoming(hub *Hub, users *UserStore, client *Client, raw []byte) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return
	}
	if len(text) > 4096 {
		text = text[:4096]
	}
	incoming := IncomingMessage{Scope: "public", Text: text}
	if err := json.Unmarshal([]byte(text), &incoming); err != nil || strings.TrimSpace(incoming.Text) == "" {
		incoming = IncomingMessage{Scope: "public", Text: text}
	}
	if incoming.Scope == "private" && incoming.To != "" {
		msg := Message{Type: "message", Name: client.User.Name, From: client.User.ID, FromTag: client.User.Tag, To: incoming.To, Text: incoming.Text, Time: now(), KeyDay: incoming.KeyDay, Private: true}
		if target, ok := users.ByID(incoming.To); ok {
			msg.ToName = target.Name
			msg.ToTag = target.Tag
		}
		hub.AddHistory(msg)
		recipients := append(hub.ClientsByID(incoming.To), hub.ClientsByID(client.User.ID)...)
		sent := make(map[*Client]bool)
		for _, recipient := range recipients {
			if sent[recipient] {
				continue
			}
			sent[recipient] = true
			hub.Send(recipient, msg)
		}
		return
	}
	msg := Message{Type: "message", Name: client.User.Name, From: client.User.ID, FromTag: client.User.Tag, Text: incoming.Text, Time: now(), KeyDay: incoming.KeyDay}
	hub.AddHistory(msg)
	hub.Broadcast(msg)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("json response error:", err)
	}
}
