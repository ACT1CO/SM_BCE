package main

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

type UserStore struct {
	mu    sync.Mutex
	path  string
	byTag map[string]User
	byID  map[string]User
}

type usersFile struct {
	Users []User `json:"users"`
}

func NewUserStore(path string) *UserStore {
	store := &UserStore{path: path, byTag: make(map[string]User), byID: make(map[string]User)}
	store.load()
	return store
}

func (s *UserStore) Register(name, tag string) (User, error) {
	name = strings.TrimSpace(name)
	tag = normalizeTag(tag)
	if name == "" {
		return User{}, errors.New("Введите имя")
	}
	if tag == "" || len([]rune(tag)) < 3 {
		return User{}, errors.New("Тег должен быть не короче 3 символов")
	}
	if len([]rune(name)) > 30 {
		name = string([]rune(name)[:30])
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, exists := s.byTag[tag]; exists {
		if strings.EqualFold(existing.Name, name) {
			return existing, nil
		}
		return User{}, errors.New("Этот тег уже занят")
	}
	user := User{ID: newID(), Name: name, Tag: tag}
	s.byTag[tag] = user
	s.byID[user.ID] = user
	s.saveLocked()
	return user, nil
}

func (s *UserStore) ByID(id string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.byID[id]
	return user, ok
}

func (s *UserStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file usersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}
	for _, user := range file.Users {
		s.byTag[user.Tag] = user
		s.byID[user.ID] = user
	}
}

func (s *UserStore) saveLocked() {
	users := make([]User, 0, len(s.byID))
	for _, user := range s.byID {
		users = append(users, user)
	}
	data, err := json.MarshalIndent(usersFile{Users: users}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0600)
}
