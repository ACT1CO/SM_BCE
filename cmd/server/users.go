package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
)

type UserStore struct {
	mu    sync.Mutex
	path  string
	db    *sql.DB
	byTag map[string]User
	byID  map[string]User
}

type usersFile struct {
	Users []User `json:"users"`
}

func NewUserStore(path string, db *sql.DB) *UserStore {
	store := &UserStore{path: path, db: db, byTag: make(map[string]User), byID: make(map[string]User)}
	if db != nil {
		store.loadFromDB()
		if len(store.byID) == 0 {
			store.loadFromJSON()
			store.saveAllToDBLocked()
		}
		return store
	}
	store.loadFromJSON()
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
	if s.db != nil {
		if err := s.insertUserLocked(user); err != nil {
			return User{}, err
		}
	}
	s.byTag[tag] = user
	s.byID[user.ID] = user
	if s.db == nil {
		s.saveJSONLocked()
	}
	return user, nil
}

func (s *UserStore) ByID(id string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.byID[id]
	return user, ok
}

func (s *UserStore) All() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]User, 0, len(s.byID))
	for _, user := range s.byID {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool {
		return strings.ToLower(users[i].Tag) < strings.ToLower(users[j].Tag)
	})
	return users
}

func (s *UserStore) loadFromJSON() {
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

func (s *UserStore) loadFromDB() {
	rows, err := s.db.Query(`SELECT id, name, tag FROM users ORDER BY lower(tag)`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Tag); err == nil {
			s.byTag[user.Tag] = user
			s.byID[user.ID] = user
		}
	}
}

func (s *UserStore) saveJSONLocked() {
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

func (s *UserStore) insertUserLocked(user User) error {
	_, err := s.db.Exec(`INSERT INTO users (id, name, tag) VALUES ($1, $2, $3)`, user.ID, user.Name, user.Tag)
	return err
}

func (s *UserStore) saveAllToDBLocked() {
	for _, user := range s.byID {
		_ = s.insertUserLocked(user)
	}
}
