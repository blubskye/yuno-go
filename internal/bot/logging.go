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

// ============================================================================
// CONFIG CACHE - In-memory caching for logging configs to reduce DB reads
// ============================================================================

type ConfigCache struct {
	configs map[string]*cachedConfig
	mu      sync.RWMutex
	ttl     time.Duration
}

type cachedConfig struct {
	config    *LoggingConfig
	expiresAt time.Time
}

func NewConfigCache(ttl time.Duration) *ConfigCache {
	return &ConfigCache{
		configs: make(map[string]*cachedConfig),
		ttl:     ttl,
	}
}

func (cc *ConfigCache) Get(guildID string) (*LoggingConfig, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	cached, exists := cc.configs[guildID]
	if !exists || time.Now().After(cached.expiresAt) {
		return nil, false
	}
	return cached.config, true
}

func (cc *ConfigCache) Set(guildID string, config *LoggingConfig) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.configs[guildID] = &cachedConfig{
		config:    config,
		expiresAt: time.Now().Add(cc.ttl),
	}
}

func (cc *ConfigCache) Invalidate(guildID string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.configs, guildID)
}

func (cc *ConfigCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.configs = make(map[string]*cachedConfig)
}

// ============================================================================
// MESSAGE CACHE BATCHER - Batches message cache writes to reduce DB transactions
// ============================================================================

type MessageCacheBatcher struct {
	bot           *Bot
	messages      []cachedMessage
	mu            sync.Mutex
	stopChan      chan struct{}
	maxBatchSize  int
	flushInterval time.Duration
}

type cachedMessage struct {
	ID        string
	GuildID   string
	ChannelID string
	AuthorID  string
	Content   string
	CreatedAt string
}

func NewMessageCacheBatcher(bot *Bot) *MessageCacheBatcher {
	return &MessageCacheBatcher{
		bot:           bot,
		messages:      make([]cachedMessage, 0, 100),
		stopChan:      make(chan struct{}),
		maxBatchSize:  100,
		flushInterval: 5 * time.Second,
	}
}

func (mcb *MessageCacheBatcher) Start() {
	go mcb.run()
}

func (mcb *MessageCacheBatcher) Stop() {
	close(mcb.stopChan)
	mcb.flush() // Final flush on shutdown
}

func (mcb *MessageCacheBatcher) Add(m *discordgo.Message) {
	if m.GuildID == "" || m.Author == nil {
		return
	}

	mcb.mu.Lock()
	mcb.messages = append(mcb.messages, cachedMessage{
		ID:        m.ID,
		GuildID:   m.GuildID,
		ChannelID: m.ChannelID,
		AuthorID:  m.Author.ID,
		Content:   m.Content,
		CreatedAt: m.Timestamp.Format(time.RFC3339),
	})

	// Flush immediately if batch is full
	if len(mcb.messages) >= mcb.maxBatchSize {
		mcb.flushLocked()
	}
	mcb.mu.Unlock()
}

func (mcb *MessageCacheBatcher) run() {
	ticker := time.NewTicker(mcb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mcb.stopChan:
			return
		case <-ticker.C:
			mcb.flush()
		}
	}
}

func (mcb *MessageCacheBatcher) flush() {
	mcb.mu.Lock()
	mcb.flushLocked()
	mcb.mu.Unlock()
}

func (mcb *MessageCacheBatcher) flushLocked() {
	if len(mcb.messages) == 0 {
		return
	}

	// Copy messages and clear buffer
	toWrite := mcb.messages
	mcb.messages = make([]cachedMessage, 0, mcb.maxBatchSize)

	// Batch insert in a single transaction
	go func(messages []cachedMessage) {
		tx, err := mcb.bot.DB.Begin()
		if err != nil {
			log.Printf("Failed to begin message cache transaction: %v", err)
			return
		}

		stmt, err := tx.Prepare(`
			INSERT OR REPLACE INTO message_cache
			(message_id, guild_id, channel_id, author_id, content, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to prepare message cache statement: %v", err)
			return
		}
		defer stmt.Close()

		for _, m := range messages {
			_, err := stmt.Exec(m.ID, m.GuildID, m.ChannelID, m.AuthorID, m.Content, m.CreatedAt)
			if err != nil {
				log.Printf("Failed to cache message %s: %v", m.ID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit message cache transaction: %v", err)
		}
	}(toWrite)
}

// ============================================================================
// EVENT LOG BATCHER - Batches voice/nickname/avatar log events
// ============================================================================

type EventLogBatcher struct {
	bot           *Bot
	events        map[string][]LogEvent // guildID -> events
	mu            sync.Mutex
	stopChan      chan struct{}
	flushInterval time.Duration
}

type LogEvent struct {
	Type      string // "voice_join", "voice_leave", "nickname", "avatar"
	UserID    string
	Username  string
	ChannelID string
	OldValue  string
	NewValue  string
	Timestamp time.Time
	Extra     map[string]string // For additional data like avatar URLs
}

func NewEventLogBatcher(bot *Bot) *EventLogBatcher {
	return &EventLogBatcher{
		bot:           bot,
		events:        make(map[string][]LogEvent),
		stopChan:      make(chan struct{}),
		flushInterval: 10 * time.Second,
	}
}

func (elb *EventLogBatcher) Start() {
	go elb.run()
}

func (elb *EventLogBatcher) Stop() {
	close(elb.stopChan)
	elb.flushAll()
}

func (elb *EventLogBatcher) AddEvent(guildID string, event LogEvent) {
	elb.mu.Lock()
	defer elb.mu.Unlock()
	elb.events[guildID] = append(elb.events[guildID], event)
}

func (elb *EventLogBatcher) run() {
	ticker := time.NewTicker(elb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-elb.stopChan:
			return
		case <-ticker.C:
			elb.flushAll()
		}
	}
}

func (elb *EventLogBatcher) flushAll() {
	elb.mu.Lock()
	defer elb.mu.Unlock()

	for guildID, events := range elb.events {
		if len(events) == 0 {
			continue
		}

		config, err := elb.bot.GetLoggingConfigCached(guildID)
		if err != nil || !config.Enabled || config.LogChannelID == "" {
			delete(elb.events, guildID)
			continue
		}

		elb.sendBatchedEvents(guildID, config, events)
		delete(elb.events, guildID)
	}
}

func (elb *EventLogBatcher) sendBatchedEvents(guildID string, config *LoggingConfig, events []LogEvent) {
	// Group events by type
	voiceJoins := []LogEvent{}
	voiceLeaves := []LogEvent{}
	nickChanges := []LogEvent{}
	avatarChanges := []LogEvent{}

	for _, e := range events {
		switch e.Type {
		case "voice_join":
			if config.MemberJoinVoice {
				voiceJoins = append(voiceJoins, e)
			}
		case "voice_leave":
			if config.MemberLeaveVoice {
				voiceLeaves = append(voiceLeaves, e)
			}
		case "nickname":
			if config.NicknameChange {
				nickChanges = append(nickChanges, e)
			}
		case "avatar":
			if config.AvatarChange {
				avatarChanges = append(avatarChanges, e)
			}
		}
	}

	// Send batched voice joins
	if len(voiceJoins) > 0 {
		elb.sendVoiceBatch(config.LogChannelID, "Members Joined Voice", 0x2ecc71, voiceJoins)
	}

	// Send batched voice leaves
	if len(voiceLeaves) > 0 {
		elb.sendVoiceBatch(config.LogChannelID, "Members Left Voice", 0xe74c3c, voiceLeaves)
	}

	// Send batched nickname changes
	if len(nickChanges) > 0 {
		elb.sendNicknameBatch(config.LogChannelID, nickChanges)
	}

	// Send batched avatar changes
	if len(avatarChanges) > 0 {
		elb.sendAvatarBatch(config.LogChannelID, avatarChanges)
	}
}

func (elb *EventLogBatcher) sendVoiceBatch(channelID, title string, color int, events []LogEvent) {
	// Group by channel
	channelMap := make(map[string][]string)
	for _, e := range events {
		channelMap[e.ChannelID] = append(channelMap[e.ChannelID], fmt.Sprintf("<@%s>", e.UserID))
	}

	var fields []*discordgo.MessageEmbedField
	for chID, users := range channelMap {
		userList := users
		extra := ""
		if len(users) > 15 {
			userList = users[:15]
			extra = fmt.Sprintf(" (+%d more)", len(users)-15)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("<#%s>", chID),
			Value:  strings.Join(userList, ", ") + extra,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**%d** voice events in the last 10 seconds", len(events)),
		Color:       color,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := elb.bot.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send batched voice events: %v", err)
	}
}

func (elb *EventLogBatcher) sendNicknameBatch(channelID string, events []LogEvent) {
	var fields []*discordgo.MessageEmbedField

	for i, e := range events {
		if i >= 10 { // Limit to 10 to avoid embed limits
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "...",
				Value:  fmt.Sprintf("+%d more nickname changes", len(events)-10),
				Inline: false,
			})
			break
		}

		oldNick := e.OldValue
		if oldNick == "" {
			oldNick = e.Username
		}
		newNick := e.NewValue
		if newNick == "" {
			newNick = e.Username
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("<@%s>", e.UserID),
			Value:  fmt.Sprintf("`%s` → `%s`", oldNick, newNick),
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Nickname Changes",
		Description: fmt.Sprintf("**%d** nickname changes in the last 10 seconds", len(events)),
		Color:       0x9b59b6,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := elb.bot.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send batched nickname changes: %v", err)
	}
}

func (elb *EventLogBatcher) sendAvatarBatch(channelID string, events []LogEvent) {
	var fields []*discordgo.MessageEmbedField

	for i, e := range events {
		if i >= 10 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "...",
				Value:  fmt.Sprintf("+%d more avatar changes", len(events)-10),
				Inline: false,
			})
			break
		}

		value := "Avatar updated"
		if newURL, ok := e.Extra["new_avatar_url"]; ok && newURL != "" {
			value = fmt.Sprintf("[New Avatar](%s)", newURL)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("<@%s>", e.UserID),
			Value:  value,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Avatar Changes",
		Description: fmt.Sprintf("**%d** avatar changes in the last 10 seconds", len(events)),
		Color:       0x1abc9c,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := elb.bot.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send batched avatar changes: %v", err)
	}
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

// GetLoggingConfig gets the logging configuration for a guild (direct DB access)
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

// GetLoggingConfigCached gets the logging configuration with caching
func (b *Bot) GetLoggingConfigCached(guildID string) (*LoggingConfig, error) {
	// Check cache first
	if b.ConfigCache != nil {
		if cached, ok := b.ConfigCache.Get(guildID); ok {
			return cached, nil
		}
	}

	// Cache miss, fetch from DB
	config, err := b.GetLoggingConfig(guildID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if b.ConfigCache != nil {
		b.ConfigCache.Set(guildID, config)
	}

	return config, nil
}

// InvalidateLoggingConfigCache invalidates the cached config for a guild
func (b *Bot) InvalidateLoggingConfigCache(guildID string) {
	if b.ConfigCache != nil {
		b.ConfigCache.Invalidate(guildID)
	}
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

// CacheMessage caches a message for potential logging (uses batcher for efficiency)
func (b *Bot) CacheMessage(m *discordgo.Message) {
	if m.GuildID == "" {
		return
	}

	// Use the batcher for efficient batched writes
	if b.MessageCacheBatcher != nil {
		b.MessageCacheBatcher.Add(m)
		return
	}

	// Fallback to direct write if batcher not available
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
