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
			last_clean TEXT,
			enabled INTEGER DEFAULT 1,
			warned INTEGER DEFAULT 0,
			custom_message TEXT DEFAULT '',
			custom_image TEXT DEFAULT '',
			PRIMARY KEY (guild_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ban_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT,
			filename TEXT,
			added_by TEXT,
			added_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS logging_config (
			guild_id TEXT PRIMARY KEY,
			log_channel_id TEXT,
			message_delete INTEGER DEFAULT 1,
			message_edit INTEGER DEFAULT 1,
			member_join_voice INTEGER DEFAULT 1,
			member_leave_voice INTEGER DEFAULT 1,
			nickname_change INTEGER DEFAULT 1,
			avatar_change INTEGER DEFAULT 1,
			presence_change INTEGER DEFAULT 1,
			presence_batch_seconds INTEGER DEFAULT 120,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS logging_channel_overrides (
			guild_id TEXT,
			channel_id TEXT,
			message_delete INTEGER DEFAULT 1,
			message_edit INTEGER DEFAULT 1,
			member_join_voice INTEGER DEFAULT 1,
			member_leave_voice INTEGER DEFAULT 1,
			nickname_change INTEGER DEFAULT 1,
			avatar_change INTEGER DEFAULT 1,
			presence_change INTEGER DEFAULT 1,
			PRIMARY KEY (guild_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS message_cache (
			message_id TEXT PRIMARY KEY,
			guild_id TEXT,
			channel_id TEXT,
			author_id TEXT,
			content TEXT,
			created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS regex_filters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			pattern TEXT NOT NULL,
			action TEXT NOT NULL,
			reason TEXT,
			enabled INTEGER DEFAULT 1,
			created_by TEXT,
			created_at TEXT,
			UNIQUE(guild_id, pattern)
		)`,
		`CREATE TABLE IF NOT EXISTS channel_regex_filters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			pattern TEXT NOT NULL,
			action TEXT NOT NULL,
			reason TEXT,
			enabled INTEGER DEFAULT 1,
			created_by TEXT,
			created_at TEXT,
			UNIQUE(guild_id, channel_id, pattern)
		)`,
		`CREATE TABLE IF NOT EXISTS spam_violations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			violation_type TEXT NOT NULL,
			reason TEXT,
			timestamp TEXT,
			moderator_id TEXT,
			action_taken TEXT
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
