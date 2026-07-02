package main

import "sync"

type Client struct {
	User    User
	Conn    WSConn
	writeMu sync.Mutex
}

type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Online bool   `json:"online,omitempty"`
}

type Message struct {
	ID       string    `json:"id,omitempty"`
	Type     string    `json:"type"`
	ClientID string    `json:"clientId,omitempty"`
	User     User      `json:"user,omitempty"`
	Users    []User    `json:"users,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Name     string    `json:"name,omitempty"`
	From     string    `json:"from,omitempty"`
	FromTag  string    `json:"fromTag,omitempty"`
	To       string    `json:"to,omitempty"`
	ToName   string    `json:"toName,omitempty"`
	ToTag    string    `json:"toTag,omitempty"`
	Text     string    `json:"text,omitempty"`
	Time     string    `json:"time,omitempty"`
	KeyDay   string    `json:"keyDay,omitempty"`
	Private  bool      `json:"private,omitempty"`
}

type IncomingMessage struct {
	Scope  string `json:"scope"`
	To     string `json:"to"`
	Text   string `json:"text"`
	KeyDay string `json:"keyDay"`
}

type RegisterRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type HistoryFile struct {
	Messages []Message `json:"messages"`
}
