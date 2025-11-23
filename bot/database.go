package bot

import (
	"database/sql"
	"log"
	"os"
)

type Database struct {
	*sql.DB
}

func NewDatabase(path string) (*Database, error) {
	os.MkdirAll("Leveling", 0755)
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	// Create all tables your Python version uses
	queries := []string{
		`CREATE TABLE IF NOT EXISTS glevel (
			guild_id TEXT,
			user_id TEXT,
			exp INTEGER DEFAULT 0,
			level INTEGER DEFAULT 0,
			enabled TEXT DEFAULT 'disabled',
			PRIMARY KEY (guild_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ranks (
			guild_id TEXT,
			role_id TEXT,
			level INTEGER,
			PRIMARY KEY (guild_id, role_id)
		)`,
		`CREATE TABLE IF NOT EXISTS welcome (
			guild_id TEXT PRIMARY KEY,
			channel_id TEXT,
			dm_enabled INTEGER DEFAULT 0,
			channel_enabled INTEGER DEFAULT 1,
			message TEXT DEFAULT 'Welcome {member} to {guild}!',
			embed_color INTEGER DEFAULT 16761035,
			image_url TEXT,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT,
			content TEXT,
			author_id TEXT,
			added_by TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS autoclean (
			guild_id TEXT,
			channel_id TEXT,
			interval_hours INTEGER,
			warning_minutes INTEGER,
			next_run TEXT,
			PRIMARY KEY (guild_id, channel_id)
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Printf("Warning: Failed to create table: %v", err)
		}
	}

	return &Database{db}, nil
}

func (d *Database) Close() {
	d.DB.Close()
}
