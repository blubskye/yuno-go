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
		// Bot-level bans (users/servers banned from using the bot)
		`CREATE TABLE IF NOT EXISTS bot_bans (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			reason TEXT,
			banned_by TEXT,
			banned_at TEXT
		)`,
		// DM forwarding configuration per guild
		`CREATE TABLE IF NOT EXISTS dm_config (
			guild_id TEXT PRIMARY KEY,
			channel_id TEXT NOT NULL,
			enabled INTEGER DEFAULT 1
		)`,
		// DM inbox storage
		`CREATE TABLE IF NOT EXISTS dm_inbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			user_tag TEXT,
			content TEXT,
			attachments TEXT,
			received_at TEXT,
			read INTEGER DEFAULT 0
		)`,
		// Voice XP tracking sessions
		`CREATE TABLE IF NOT EXISTS voice_sessions (
			guild_id TEXT,
			user_id TEXT,
			channel_id TEXT,
			joined_at TEXT,
			PRIMARY KEY (guild_id, user_id)
		)`,
		// Voice XP configuration per guild
		`CREATE TABLE IF NOT EXISTS voice_xp_config (
			guild_id TEXT PRIMARY KEY,
			enabled INTEGER DEFAULT 0,
			xp_rate INTEGER DEFAULT 10,
			interval_seconds INTEGER DEFAULT 300,
			ignore_afk INTEGER DEFAULT 1
		)`,
		// Guild prefixes
		`CREATE TABLE IF NOT EXISTS guild_prefixes (
			guild_id TEXT PRIMARY KEY,
			prefix TEXT NOT NULL
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

// Bot Ban Methods

// IsBotBanned checks if a user or server is banned from using the bot
func (d *Database) IsBotBanned(userID, guildID string) (bool, string) {
	// Check user ban
	var reason string
	err := d.QueryRow("SELECT reason FROM bot_bans WHERE id = ? AND type = 'user'", userID).Scan(&reason)
	if err == nil {
		return true, reason
	}

	// Check server ban if guildID provided
	if guildID != "" {
		err = d.QueryRow("SELECT reason FROM bot_bans WHERE id = ? AND type = 'server'", guildID).Scan(&reason)
		if err == nil {
			return true, reason
		}
	}

	return false, ""
}

// AddBotBan adds a bot-level ban
func (d *Database) AddBotBan(id, banType, reason, bannedBy string) error {
	_, err := d.Exec(
		"INSERT OR REPLACE INTO bot_bans (id, type, reason, banned_by, banned_at) VALUES (?, ?, ?, ?, datetime('now'))",
		id, banType, reason, bannedBy,
	)
	return err
}

// RemoveBotBan removes a bot-level ban
func (d *Database) RemoveBotBan(id string) error {
	_, err := d.Exec("DELETE FROM bot_bans WHERE id = ?", id)
	return err
}

// GetBotBans retrieves all bot-level bans, optionally filtered by type
func (d *Database) GetBotBans(banType string) ([]map[string]string, error) {
	var rows *sql.Rows
	var err error

	if banType == "" {
		rows, err = d.Query("SELECT id, type, reason, banned_by, banned_at FROM bot_bans ORDER BY banned_at DESC")
	} else {
		rows, err = d.Query("SELECT id, type, reason, banned_by, banned_at FROM bot_bans WHERE type = ? ORDER BY banned_at DESC", banType)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []map[string]string
	for rows.Next() {
		var id, t, reason, bannedBy, bannedAt string
		if err := rows.Scan(&id, &t, &reason, &bannedBy, &bannedAt); err != nil {
			continue
		}
		bans = append(bans, map[string]string{
			"id":        id,
			"type":      t,
			"reason":    reason,
			"banned_by": bannedBy,
			"banned_at": bannedAt,
		})
	}
	return bans, nil
}

// DM Config Methods

// GetDMConfig retrieves DM forwarding config for a guild
func (d *Database) GetDMConfig(guildID string) (string, bool, error) {
	var channelID string
	var enabled int
	err := d.QueryRow("SELECT channel_id, enabled FROM dm_config WHERE guild_id = ?", guildID).Scan(&channelID, &enabled)
	if err != nil {
		return "", false, err
	}
	return channelID, enabled == 1, nil
}

// SetDMConfig sets DM forwarding config for a guild
func (d *Database) SetDMConfig(guildID, channelID string) error {
	_, err := d.Exec(
		"INSERT OR REPLACE INTO dm_config (guild_id, channel_id, enabled) VALUES (?, ?, 1)",
		guildID, channelID,
	)
	return err
}

// RemoveDMConfig removes DM forwarding config for a guild
func (d *Database) RemoveDMConfig(guildID string) error {
	_, err := d.Exec("DELETE FROM dm_config WHERE guild_id = ?", guildID)
	return err
}

// GetAllDMConfigs retrieves all DM forwarding configs
func (d *Database) GetAllDMConfigs() ([]map[string]string, error) {
	rows, err := d.Query("SELECT guild_id, channel_id, enabled FROM dm_config WHERE enabled = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []map[string]string
	for rows.Next() {
		var guildID, channelID string
		var enabled int
		if err := rows.Scan(&guildID, &channelID, &enabled); err != nil {
			continue
		}
		configs = append(configs, map[string]string{
			"guild_id":   guildID,
			"channel_id": channelID,
		})
	}
	return configs, nil
}

// DM Inbox Methods

// SaveDM saves a DM to the inbox
func (d *Database) SaveDM(userID, userTag, content, attachments string) (int64, error) {
	result, err := d.Exec(
		"INSERT INTO dm_inbox (user_id, user_tag, content, attachments, received_at, read) VALUES (?, ?, ?, ?, datetime('now'), 0)",
		userID, userTag, content, attachments,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetDMs retrieves DMs from inbox
func (d *Database) GetDMs(limit int) ([]map[string]interface{}, error) {
	rows, err := d.Query(
		"SELECT id, user_id, user_tag, content, attachments, received_at, read FROM dm_inbox ORDER BY received_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dms []map[string]interface{}
	for rows.Next() {
		var id int64
		var userID, userTag, content, attachments, receivedAt string
		var read int
		if err := rows.Scan(&id, &userID, &userTag, &content, &attachments, &receivedAt, &read); err != nil {
			continue
		}
		dms = append(dms, map[string]interface{}{
			"id":          id,
			"user_id":     userID,
			"user_tag":    userTag,
			"content":     content,
			"attachments": attachments,
			"received_at": receivedAt,
			"read":        read == 1,
		})
	}
	return dms, nil
}

// GetDMsByUser retrieves DMs from a specific user
func (d *Database) GetDMsByUser(userID string, limit int) ([]map[string]interface{}, error) {
	rows, err := d.Query(
		"SELECT id, user_id, user_tag, content, attachments, received_at, read FROM dm_inbox WHERE user_id = ? ORDER BY received_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dms []map[string]interface{}
	for rows.Next() {
		var id int64
		var uID, userTag, content, attachments, receivedAt string
		var read int
		if err := rows.Scan(&id, &uID, &userTag, &content, &attachments, &receivedAt, &read); err != nil {
			continue
		}
		dms = append(dms, map[string]interface{}{
			"id":          id,
			"user_id":     uID,
			"user_tag":    userTag,
			"content":     content,
			"attachments": attachments,
			"received_at": receivedAt,
			"read":        read == 1,
		})
	}
	return dms, nil
}

// MarkDMRead marks a DM as read
func (d *Database) MarkDMRead(id int64) error {
	_, err := d.Exec("UPDATE dm_inbox SET read = 1 WHERE id = ?", id)
	return err
}

// GetUnreadDMCount returns count of unread DMs
func (d *Database) GetUnreadDMCount() (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM dm_inbox WHERE read = 0").Scan(&count)
	return count, err
}

// ClearOldDMs removes DMs older than specified days
func (d *Database) ClearOldDMs(days int) (int64, error) {
	result, err := d.Exec(
		"DELETE FROM dm_inbox WHERE received_at < datetime('now', '-' || ? || ' days')",
		days,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetDMByID retrieves a specific DM by inbox ID
func (d *Database) GetDMByID(id int64) (map[string]interface{}, error) {
	var userID, userTag, content, attachments, receivedAt string
	var read int
	err := d.QueryRow(
		"SELECT user_id, user_tag, content, attachments, received_at, read FROM dm_inbox WHERE id = ?",
		id,
	).Scan(&userID, &userTag, &content, &attachments, &receivedAt, &read)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"user_id":     userID,
		"user_tag":    userTag,
		"content":     content,
		"attachments": attachments,
		"received_at": receivedAt,
		"read":        read == 1,
	}, nil
}

// Voice XP Methods

// GetVoiceXPConfig retrieves voice XP config for a guild
func (d *Database) GetVoiceXPConfig(guildID string) (bool, int, int, bool, error) {
	var enabled, ignoreAFK int
	var xpRate, interval int
	err := d.QueryRow(
		"SELECT enabled, xp_rate, interval_seconds, ignore_afk FROM voice_xp_config WHERE guild_id = ?",
		guildID,
	).Scan(&enabled, &xpRate, &interval, &ignoreAFK)
	if err != nil {
		// Return defaults if not configured
		return false, 10, 300, true, nil
	}
	return enabled == 1, xpRate, interval, ignoreAFK == 1, nil
}

// SetVoiceXPConfig sets voice XP config for a guild
func (d *Database) SetVoiceXPConfig(guildID string, enabled bool, xpRate, interval int, ignoreAFK bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	ignoreAFKInt := 0
	if ignoreAFK {
		ignoreAFKInt = 1
	}
	_, err := d.Exec(
		"INSERT OR REPLACE INTO voice_xp_config (guild_id, enabled, xp_rate, interval_seconds, ignore_afk) VALUES (?, ?, ?, ?, ?)",
		guildID, enabledInt, xpRate, interval, ignoreAFKInt,
	)
	return err
}

// SaveVoiceSession saves a voice session start
func (d *Database) SaveVoiceSession(guildID, userID, channelID string) error {
	_, err := d.Exec(
		"INSERT OR REPLACE INTO voice_sessions (guild_id, user_id, channel_id, joined_at) VALUES (?, ?, ?, datetime('now'))",
		guildID, userID, channelID,
	)
	return err
}

// GetVoiceSession retrieves a voice session
func (d *Database) GetVoiceSession(guildID, userID string) (string, string, error) {
	var channelID, joinedAt string
	err := d.QueryRow(
		"SELECT channel_id, joined_at FROM voice_sessions WHERE guild_id = ? AND user_id = ?",
		guildID, userID,
	).Scan(&channelID, &joinedAt)
	return channelID, joinedAt, err
}

// RemoveVoiceSession removes a voice session
func (d *Database) RemoveVoiceSession(guildID, userID string) error {
	_, err := d.Exec("DELETE FROM voice_sessions WHERE guild_id = ? AND user_id = ?", guildID, userID)
	return err
}

// GetAllVoiceSessions retrieves all active voice sessions
func (d *Database) GetAllVoiceSessions() ([]map[string]string, error) {
	rows, err := d.Query("SELECT guild_id, user_id, channel_id, joined_at FROM voice_sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []map[string]string
	for rows.Next() {
		var guildID, userID, channelID, joinedAt string
		if err := rows.Scan(&guildID, &userID, &channelID, &joinedAt); err != nil {
			continue
		}
		sessions = append(sessions, map[string]string{
			"guild_id":   guildID,
			"user_id":    userID,
			"channel_id": channelID,
			"joined_at":  joinedAt,
		})
	}
	return sessions, nil
}

// Guild Prefix Methods

// GetGuildPrefix retrieves custom prefix for a guild
func (d *Database) GetGuildPrefix(guildID string) (string, error) {
	var prefix string
	err := d.QueryRow("SELECT prefix FROM guild_prefixes WHERE guild_id = ?", guildID).Scan(&prefix)
	return prefix, err
}

// SetGuildPrefix sets custom prefix for a guild
func (d *Database) SetGuildPrefix(guildID, prefix string) error {
	_, err := d.Exec(
		"INSERT OR REPLACE INTO guild_prefixes (guild_id, prefix) VALUES (?, ?)",
		guildID, prefix,
	)
	return err
}
