package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	hub := NewHub("chat-history.json")
	keys := NewKeyManager("chat-key.json")
	users := NewUserStore("users.json")
	keys.Current()

	fs := http.FileServer(http.Dir(webDir()))
	http.HandleFunc("/key", keyHandler(keys))
	http.HandleFunc("/register", registerHandler(users))
	http.HandleFunc("/ws", wsHandler(hub, users))
	http.Handle("/", fs)

	addr := ":8080"
	log.Println("Соцсети-ВСЁ! started on http://localhost" + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func webDir() string {
	candidates := []string{"web", "../../web"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "web"
}
