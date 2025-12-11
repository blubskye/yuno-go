package bot

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"
)

// ============================================================================
// XP BATCHER - Batches XP updates to reduce database transactions
// ============================================================================

// XPBatcher batches XP updates to reduce DB transactions
type XPBatcher struct {
	bot           *Bot
	pending       map[string]*PendingXP // key: "guildID:userID"
	mu            sync.Mutex
	stopChan      chan struct{}
	flushInterval time.Duration
	maxBatchSize  int
}

// PendingXP holds accumulated XP for a user
type PendingXP struct {
	GuildID   string
	UserID    string
	ChannelID string // Last channel (for level-up message)
	XP        int
	AddedAt   time.Time
}

// NewXPBatcher creates a new XP batcher
func NewXPBatcher(bot *Bot) *XPBatcher {
	return &XPBatcher{
		bot:           bot,
		pending:       make(map[string]*PendingXP),
		stopChan:      make(chan struct{}),
		flushInterval: 10 * time.Second,
		maxBatchSize:  200,
	}
}

// Start begins the XP batching loop
func (xb *XPBatcher) Start() {
	go xb.run()
}

// Stop stops the XP batcher and flushes remaining XP
func (xb *XPBatcher) Stop() {
	close(xb.stopChan)
	xb.flush() // Final flush
}

// AddXP adds XP to the batch for a user
func (xb *XPBatcher) AddXP(guildID, userID, channelID string, xp int) {
	key := guildID + ":" + userID

	xb.mu.Lock()
	if existing, ok := xb.pending[key]; ok {
		existing.XP += xp
		existing.ChannelID = channelID // Update to latest channel
	} else {
		xb.pending[key] = &PendingXP{
			GuildID:   guildID,
			UserID:    userID,
			ChannelID: channelID,
			XP:        xp,
			AddedAt:   time.Now(),
		}
	}

	// Flush if batch is full
	if len(xb.pending) >= xb.maxBatchSize {
		xb.flushLocked()
	}
	xb.mu.Unlock()
}

func (xb *XPBatcher) run() {
	ticker := time.NewTicker(xb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-xb.stopChan:
			return
		case <-ticker.C:
			xb.flush()
		}
	}
}

func (xb *XPBatcher) flush() {
	xb.mu.Lock()
	xb.flushLocked()
	xb.mu.Unlock()
}

func (xb *XPBatcher) flushLocked() {
	if len(xb.pending) == 0 {
		return
	}

	// Copy and clear
	toProcess := xb.pending
	xb.pending = make(map[string]*PendingXP)

	// Process in background
	go xb.processBatch(toProcess)
}

func (xb *XPBatcher) processBatch(batch map[string]*PendingXP) {
	defer RecoverFromPanic("XPBatcher.processBatch")

	tx, err := xb.bot.DB.Begin()
	if err != nil {
		log.Printf("[XP Batcher] Failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Prepare statements
	selectStmt, err := tx.Prepare(`SELECT exp, level, enabled FROM glevel WHERE guild_id = ? AND user_id = ?`)
	if err != nil {
		log.Printf("[XP Batcher] Failed to prepare select: %v", err)
		return
	}
	defer selectStmt.Close()

	insertStmt, err := tx.Prepare(`INSERT OR REPLACE INTO glevel (guild_id, user_id, exp, level, enabled) VALUES (?, ?, ?, ?, 'enabled')`)
	if err != nil {
		log.Printf("[XP Batcher] Failed to prepare insert: %v", err)
		return
	}
	defer insertStmt.Close()

	levelUps := []levelUpNotification{}

	for _, p := range batch {
		var currentXP, level int
		var enabled string

		err := selectStmt.QueryRow(p.GuildID, p.UserID).Scan(&currentXP, &level, &enabled)
		if err != nil {
			// New user
			currentXP = 0
			level = 0
			enabled = "enabled"
		}

		if enabled != "enabled" {
			continue
		}

		newXP := currentXP + p.XP
		newLevel := int(math.Floor((math.Sqrt(1 + 8*float64(newXP)/50) - 1) / 2))

		_, err = insertStmt.Exec(p.GuildID, p.UserID, newXP, newLevel)
		if err != nil {
			log.Printf("[XP Batcher] Failed to update XP for %s: %v", p.UserID, err)
			continue
		}

		// Track level ups for notifications
		if newLevel > level && p.ChannelID != "" {
			levelUps = append(levelUps, levelUpNotification{
				GuildID:   p.GuildID,
				UserID:    p.UserID,
				ChannelID: p.ChannelID,
				OldLevel:  level,
				NewLevel:  newLevel,
			})
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[XP Batcher] Failed to commit: %v", err)
		return
	}

	// Send level-up notifications and check roles (outside transaction)
	for _, lu := range levelUps {
		go xb.handleLevelUp(lu)
	}

	DebugLog("[XP Batcher] Processed %d XP updates, %d level-ups", len(batch), len(levelUps))
}

type levelUpNotification struct {
	GuildID   string
	UserID    string
	ChannelID string
	OldLevel  int
	NewLevel  int
}

func (xb *XPBatcher) handleLevelUp(lu levelUpNotification) {
	defer RecoverFromPanic(fmt.Sprintf("handleLevelUp(user=%s)", lu.UserID))

	user, err := xb.bot.Session.User(lu.UserID)
	if err == nil {
		xb.bot.Session.ChannelMessageSend(lu.ChannelID,
			user.Mention()+" just reached **Level "+strconv.Itoa(lu.NewLevel)+"**! ♡")
	}

	// Check for role rewards
	xb.bot.checkLevelRoles(lu.GuildID, lu.UserID, lu.NewLevel)
}

// ============================================================================
// VOICE XP CONFIG CACHE - Caches voice XP configs to reduce DB reads
// ============================================================================

type VoiceXPConfigCache struct {
	configs map[string]*cachedVoiceConfig
	mu      sync.RWMutex
	ttl     time.Duration
}

type cachedVoiceConfig struct {
	Enabled   bool
	XPRate    int
	Interval  int
	IgnoreAFK bool
	ExpiresAt time.Time
}

func NewVoiceXPConfigCache(ttl time.Duration) *VoiceXPConfigCache {
	return &VoiceXPConfigCache{
		configs: make(map[string]*cachedVoiceConfig),
		ttl:     ttl,
	}
}

func (vc *VoiceXPConfigCache) Get(guildID string) (enabled bool, xpRate, interval int, ignoreAFK bool, ok bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	cached, exists := vc.configs[guildID]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return false, 0, 0, false, false
	}
	return cached.Enabled, cached.XPRate, cached.Interval, cached.IgnoreAFK, true
}

func (vc *VoiceXPConfigCache) Set(guildID string, enabled bool, xpRate, interval int, ignoreAFK bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.configs[guildID] = &cachedVoiceConfig{
		Enabled:   enabled,
		XPRate:    xpRate,
		Interval:  interval,
		IgnoreAFK: ignoreAFK,
		ExpiresAt: time.Now().Add(vc.ttl),
	}
}

func (vc *VoiceXPConfigCache) Invalidate(guildID string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	delete(vc.configs, guildID)
}

// GetVoiceXPConfigCached gets voice XP config with caching
func (b *Bot) GetVoiceXPConfigCached(guildID string) (enabled bool, xpRate, interval int, ignoreAFK bool, err error) {
	// Check cache first
	if b.VoiceXPConfigCache != nil {
		if enabled, xpRate, interval, ignoreAFK, ok := b.VoiceXPConfigCache.Get(guildID); ok {
			return enabled, xpRate, interval, ignoreAFK, nil
		}
	}

	// Cache miss, fetch from DB
	enabled, xpRate, interval, ignoreAFK, err = b.DB.GetVoiceXPConfig(guildID)
	if err != nil {
		return false, 0, 0, false, err
	}

	// Store in cache
	if b.VoiceXPConfigCache != nil {
		b.VoiceXPConfigCache.Set(guildID, enabled, xpRate, interval, ignoreAFK)
	}

	return enabled, xpRate, interval, ignoreAFK, nil
}
