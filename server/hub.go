package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients     map[*Client]bool
	history     []Message
	historyPath string
	db          *sql.DB
	users       *UserStore
	mu          sync.Mutex
}

func NewHub(historyPath string, db *sql.DB) *Hub {
	hub := &Hub{clients: make(map[*Client]bool), historyPath: historyPath, db: db}
	hub.loadHistory()
	return hub
}

func (h *Hub) SetUsers(users *UserStore) {
	h.users = users
}

func (h *Hub) loadHistory() {
	if h.db != nil {
		h.loadHistoryFromDB()
		if len(h.history) > 0 {
			return
		}
	}

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

	if h.db != nil {
		for _, msg := range h.history {
			if msg.ID == "" {
				msg.ID = newID()
			}
			_ = h.insertMessage(msg)
		}
	}
}

func (h *Hub) loadHistoryFromDB() {
	rows, err := h.db.Query(`
SELECT id, type, name, from_id, from_tag, to_id, to_name, to_tag, text, sent_at, key_day, private, deleted, read_by
FROM messages
ORDER BY created_at, sent_at
LIMIT 1000`)
	if err != nil {
		log.Println("history db load error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var name, from, fromTag, to, toName, toTag, text, sentAt, keyDay, readBy sql.NullString
		if err := rows.Scan(&msg.ID, &msg.Type, &name, &from, &fromTag, &to, &toName, &toTag, &text, &sentAt, &keyDay, &msg.Private, &msg.Deleted, &readBy); err != nil {
			continue
		}
		msg.Name = name.String
		msg.From = from.String
		msg.FromTag = fromTag.String
		msg.To = to.String
		msg.ToName = toName.String
		msg.ToTag = toTag.String
		msg.Text = text.String
		msg.Time = sentAt.String
		msg.KeyDay = keyDay.String
		msg.ReadBy = decodeReadBy(readBy.String)
		h.history = append(h.history, msg)
	}
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

func (h *Hub) AddHistory(msg Message) Message {
	if msg.ID == "" {
		msg.ID = newID()
	}

	h.mu.Lock()
	h.history = append(h.history, msg)
	if h.db != nil {
		if err := h.insertMessage(msg); err != nil {
			log.Println("history db save error:", err)
		}
	} else {
		h.saveHistoryLocked()
	}
	h.mu.Unlock()
	return msg
}

func (h *Hub) insertMessage(msg Message) error {
	_, err := h.db.Exec(`
INSERT INTO messages (id, type, name, from_id, from_tag, to_id, to_name, to_tag, text, sent_at, key_day, private, deleted, read_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (id) DO NOTHING`,
		msg.ID, msg.Type, msg.Name, msg.From, msg.FromTag, msg.To, msg.ToName, msg.ToTag, msg.Text, msg.Time, msg.KeyDay, msg.Private, msg.Deleted, encodeReadBy(msg.ReadBy))
	return err
}

func (h *Hub) DeleteMessage(id, userID string) (Message, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.history {
		msg := &h.history[i]
		if msg.ID != id || msg.Type != "message" || msg.From != userID || msg.Deleted {
			continue
		}
		msg.Deleted = true
		msg.Text = ""
		msg.KeyDay = ""
		updated := *msg
		if h.db != nil {
			if err := h.updateMessageDeleted(updated.ID); err != nil {
				log.Println("history db delete mark error:", err)
			}
		} else {
			h.saveHistoryLocked()
		}
		return updated, true
	}
	return Message{}, false
}

func (h *Hub) MarkRead(ids []string, userID string) []Message {
	if len(ids) == 0 {
		return nil
	}
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			idSet[id] = true
		}
	}

	h.mu.Lock()
	changed := false
	updated := make([]Message, 0)
	for i := range h.history {
		msg := &h.history[i]
		if !idSet[msg.ID] || msg.Type != "message" || msg.Deleted || msg.From == userID {
			continue
		}
		if msg.Private && msg.To != userID {
			continue
		}
		if !msg.Private {
			continue
		}
		if hasString(msg.ReadBy, userID) {
			continue
		}
		msg.ReadBy = append(msg.ReadBy, userID)
		changed = true
		updated = append(updated, *msg)
		if h.db != nil {
			if err := h.updateMessageReadBy(msg.ID, msg.ReadBy); err != nil {
				log.Println("history db read mark error:", err)
			}
		}
	}
	if changed && h.db == nil {
		h.saveHistoryLocked()
	}
	h.mu.Unlock()

	return updated
}

func (h *Hub) updateMessageDeleted(id string) error {
	_, err := h.db.Exec(`UPDATE messages SET deleted = true, text = '', key_day = '' WHERE id = $1`, id)
	return err
}

func (h *Hub) updateMessageReadBy(id string, readBy []string) error {
	_, err := h.db.Exec(`UPDATE messages SET read_by = $2 WHERE id = $1`, id, encodeReadBy(readBy))
	return err
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

func (h *Hub) SendMessageEvent(msg Message) {
	if !msg.Private {
		h.Broadcast(msg)
		return
	}
	recipients := append(h.ClientsByID(msg.To), h.ClientsByID(msg.From)...)
	sent := make(map[*Client]bool)
	for _, recipient := range recipients {
		if sent[recipient] {
			continue
		}
		sent[recipient] = true
		h.Send(recipient, msg)
	}
}

func (h *Hub) BroadcastUsers() {
	h.mu.Lock()
	online := make(map[string]bool)
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
		online[client.User.ID] = true
	}
	h.mu.Unlock()

	users := []User{}
	if h.users != nil {
		users = h.users.All()
	}
	for i := range users {
		users[i].Online = online[users[i].ID]
	}

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

func hasString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func encodeReadBy(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ",")
}

func decodeReadBy(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
