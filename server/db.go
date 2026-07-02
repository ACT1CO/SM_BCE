package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func openDBFromEnv() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Println("postgres open error:", err)
		return nil
	}
	if err := db.Ping(); err != nil {
		log.Println("postgres ping error:", err)
		_ = db.Close()
		return nil
	}
	if err := migrateDB(db); err != nil {
		log.Println("postgres migration error:", err)
		_ = db.Close()
		return nil
	}
	log.Println("PostgreSQL storage enabled")
	return db
}

func migrateDB(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	tag TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	name TEXT,
	from_id TEXT,
	from_tag TEXT,
	to_id TEXT,
	to_name TEXT,
	to_tag TEXT,
	text TEXT,
	sent_at TEXT,
	key_day TEXT,
	private BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS messages_created_at_idx ON messages (created_at);
CREATE INDEX IF NOT EXISTS messages_private_to_idx ON messages (private, to_id);
CREATE INDEX IF NOT EXISTS messages_private_from_idx ON messages (private, from_id);
`)
	return err
}
