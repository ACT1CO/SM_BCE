package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type KeyManager struct {
	mu   sync.Mutex
	path string
	day  string
	key  string
	keys map[string]string
}

type storedKeys struct {
	Day  string            `json:"day"`
	Key  string            `json:"key"`
	Keys map[string]string `json:"keys"`
}

func NewKeyManager(path string) *KeyManager {
	return &KeyManager{path: path, keys: make(map[string]string)}
}

func (k *KeyManager) Current() (string, string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	day := time.Now().Format("2006-01-02")
	if k.key == "" {
		k.load()
	}
	if k.keys == nil {
		k.keys = make(map[string]string)
	}
	if k.key == "" || k.day != day {
		if existing := k.keys[day]; existing != "" {
			k.key = existing
		} else {
			k.key = randomKey()
			k.keys[day] = k.key
		}
		k.day = day
		k.save()
		log.Println("chat encryption key rotated for", day)
	}

	return k.day, k.key
}

func (k *KeyManager) Snapshot() (string, string, map[string]string) {
	day, key := k.Current()
	k.mu.Lock()
	defer k.mu.Unlock()

	keys := make(map[string]string, len(k.keys))
	for keyDay, value := range k.keys {
		keys[keyDay] = value
	}
	if keys[day] == "" {
		keys[day] = key
	}
	return day, key, keys
}

func (k *KeyManager) load() {
	data, err := os.ReadFile(k.path)
	if err != nil {
		return
	}
	var saved storedKeys
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Println("stored key parse error:", err)
		return
	}
	k.day = saved.Day
	k.key = saved.Key
	k.keys = saved.Keys
	if k.keys == nil {
		k.keys = make(map[string]string)
	}
	if k.day != "" && k.key != "" {
		k.keys[k.day] = k.key
	}
}

func (k *KeyManager) save() {
	data, err := json.MarshalIndent(storedKeys{Day: k.day, Key: k.key, Keys: k.keys}, "", "  ")
	if err != nil {
		log.Println("stored key marshal error:", err)
		return
	}
	if err := os.WriteFile(k.path, data, 0600); err != nil {
		log.Println("stored key save error:", err)
	}
}

func randomKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("random key error:", err)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
