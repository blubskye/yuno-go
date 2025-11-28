package bot

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// LoggingConfig holds the logging configuration for a guild
type LoggingConfig struct {
	GuildID              string
	LogChannelID         string
	MessageDelete        bool
	MessageEdit          bool
	MemberJoinVoice      bool
	MemberLeaveVoice     bool
	NicknameChange       bool
	AvatarChange         bool
	PresenceChange       bool
	PresenceBatchSeconds int
	Enabled              bool
}

// PresenceBatcher batches presence changes to avoid spam
type PresenceBatcher struct {
	mu              sync.Mutex
	bot             *Bot
	changes         map[string][]PresenceChange // guildID -> changes
	stopChan        chan struct{}
	defaultInterval time.Duration
}

type PresenceChange struct {
	UserID    string
	Username  string
	OldStatus string
	NewStatus string
	Timestamp time.Time
}

// NewPresenceBatcher creates a new presence batcher
func NewPresenceBatcher(bot *Bot) *PresenceBatcher {
	return &PresenceBatcher{
		bot:             bot,
		changes:         make(map[string][]PresenceChange),
		stopChan:        make(chan struct{}),
		defaultInterval: 2 * time.Minute,
	}
}

// Start begins the batching goroutine
func (pb *PresenceBatcher) Start() {
	go pb.run()
}

// Stop stops the batching goroutine
func (pb *PresenceBatcher) Stop() {
	close(pb.stopChan)
}

// AddChange adds a presence change to the batch
func (pb *PresenceBatcher) AddChange(guildID string, change PresenceChange) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.changes[guildID] = append(pb.changes[guildID], change)
}

// run is the main batching loop
func (pb *PresenceBatcher) run() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-pb.stopChan:
			return
		case <-ticker.C:
			pb.flushAll()
		}
	}
}

// flushAll flushes all batched changes
func (pb *PresenceBatcher) flushAll() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	for guildID, changes := range pb.changes {
		if len(changes) == 0 {
			continue
		}

		// Get the batch interval for this guild
		interval := pb.getBatchInterval(guildID)

		// Check if oldest change is older than the interval
		if time.Since(changes[0].Timestamp) < interval {
			continue
		}

		// Get logging config
		config, err := pb.bot.GetLoggingConfig(guildID)
		if err != nil || !config.Enabled || !config.PresenceChange || config.LogChannelID == "" {
			delete(pb.changes, guildID)
			continue
		}

		// Build the message
		pb.sendBatchedPresences(guildID, config.LogChannelID, changes)

		// Clear the changes for this guild
		delete(pb.changes, guildID)
	}
}

// getBatchInterval gets the batch interval for a guild
func (pb *PresenceBatcher) getBatchInterval(guildID string) time.Duration {
	config, err := pb.bot.GetLoggingConfig(guildID)
	if err != nil {
		return pb.defaultInterval
	}
	return time.Duration(config.PresenceBatchSeconds) * time.Second
}

// sendBatchedPresences sends batched presence changes
func (pb *PresenceBatcher) sendBatchedPresences(guildID, logChannelID string, changes []PresenceChange) {
	if len(changes) == 0 {
		return
	}

	// Group changes by status transition
	statusMap := make(map[string][]string)
	for _, change := range changes {
		key := fmt.Sprintf("%s → %s", change.OldStatus, change.NewStatus)
		statusMap[key] = append(statusMap[key], change.Username)
	}

	// Build embed fields
	var fields []*discordgo.MessageEmbedField
	for transition, users := range statusMap {
		// Limit to first 20 users per transition to avoid hitting embed limits
		userList := users
		extra := ""
		if len(users) > 20 {
			userList = users[:20]
			extra = fmt.Sprintf(" (+%d more)", len(users)-20)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   transition,
			Value:  strings.Join(userList, ", ") + extra,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Presence Changes",
		Description: fmt.Sprintf("**%d** presence changes in the last %s", len(changes), pb.getBatchInterval(guildID)),
		Color:       0x3498db, // Blue
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := pb.bot.Session.ChannelMessageSendEmbed(logChannelID, embed)
	if err != nil {
		log.Printf("Failed to send batched presence changes: %v", err)
	}
}

// GetLoggingConfig gets the logging configuration for a guild
func (b *Bot) GetLoggingConfig(guildID string) (*LoggingConfig, error) {
	var config LoggingConfig
	var (
		msgDel, msgEdit, joinVoice, leaveVoice, nick, avatar, presence, enabled int
	)

	err := b.DB.QueryRow(`
		SELECT log_channel_id, message_delete, message_edit, member_join_voice,
		       member_leave_voice, nickname_change, avatar_change, presence_change,
		       presence_batch_seconds, enabled
		FROM logging_config WHERE guild_id = ?`, guildID).Scan(
		&config.LogChannelID, &msgDel, &msgEdit, &joinVoice, &leaveVoice,
		&nick, &avatar, &presence, &config.PresenceBatchSeconds, &enabled,
	)

	if err == sql.ErrNoRows {
		// Return default config
		return &LoggingConfig{
			GuildID:              guildID,
			MessageDelete:        true,
			MessageEdit:          true,
			MemberJoinVoice:      true,
			MemberLeaveVoice:     true,
			NicknameChange:       true,
			AvatarChange:         true,
			PresenceChange:       true,
			PresenceBatchSeconds: 120,
			Enabled:              false,
		}, nil
	} else if err != nil {
		return nil, err
	}

	config.GuildID = guildID
	config.MessageDelete = msgDel == 1
	config.MessageEdit = msgEdit == 1
	config.MemberJoinVoice = joinVoice == 1
	config.MemberLeaveVoice = leaveVoice == 1
	config.NicknameChange = nick == 1
	config.AvatarChange = avatar == 1
	config.PresenceChange = presence == 1
	config.Enabled = enabled == 1

	return &config, nil
}

// IsChannelLoggingEnabled checks if a specific logging type is enabled for a channel
func (b *Bot) IsChannelLoggingEnabled(guildID, channelID, logType string) (bool, error) {
	// First check if there's a channel override
	var override int
	query := fmt.Sprintf("SELECT %s FROM logging_channel_overrides WHERE guild_id = ? AND channel_id = ?", logType)
	err := b.DB.QueryRow(query, guildID, channelID).Scan(&override)

	if err == nil {
		// Override exists
		return override == 1, nil
	}

	// No override, use guild config
	config, err := b.GetLoggingConfig(guildID)
	if err != nil {
		return false, err
	}

	switch logType {
	case "message_delete":
		return config.MessageDelete, nil
	case "message_edit":
		return config.MessageEdit, nil
	case "member_join_voice":
		return config.MemberJoinVoice, nil
	case "member_leave_voice":
		return config.MemberLeaveVoice, nil
	case "nickname_change":
		return config.NicknameChange, nil
	case "avatar_change":
		return config.AvatarChange, nil
	case "presence_change":
		return config.PresenceChange, nil
	}

	return false, nil
}

// CacheMessage caches a message for potential logging
func (b *Bot) CacheMessage(m *discordgo.Message) {
	if m.GuildID == "" {
		return
	}

	_, err := b.DB.Exec(`
		INSERT OR REPLACE INTO message_cache (message_id, guild_id, channel_id, author_id, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		m.ID, m.GuildID, m.ChannelID, m.Author.ID, m.Content, m.Timestamp.Format(time.RFC3339),
	)

	if err != nil {
		log.Printf("Failed to cache message: %v", err)
	}
}

// GetCachedMessage retrieves a cached message
func (b *Bot) GetCachedMessage(messageID string) (*discordgo.Message, error) {
	var m discordgo.Message
	var createdAt string

	err := b.DB.QueryRow(`
		SELECT message_id, guild_id, channel_id, author_id, content, created_at
		FROM message_cache WHERE message_id = ?`, messageID).Scan(
		&m.ID, &m.GuildID, &m.ChannelID, &m.Author.ID, &m.Content, &createdAt,
	)

	if err != nil {
		return nil, err
	}

	m.Timestamp, _ = time.Parse(time.RFC3339, createdAt)
	return &m, nil
}

// CleanOldMessageCache cleans up old messages from the cache (older than 7 days)
func (b *Bot) CleanOldMessageCache() {
	cutoff := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	_, err := b.DB.Exec("DELETE FROM message_cache WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("Failed to clean message cache: %v", err)
	}
}
