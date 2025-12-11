package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AutoCleanWorker manages automatic channel cleaning
type AutoCleanWorker struct {
	bot           *Bot
	stopChan      chan bool
	ticker        *time.Ticker
	cleaningLocks sync.Map // Prevents concurrent cleans of the same channel
}

// NewAutoCleanWorker creates a new auto-clean worker
func NewAutoCleanWorker(bot *Bot) *AutoCleanWorker {
	return &AutoCleanWorker{
		bot:      bot,
		stopChan: make(chan bool),
	}
}

// Start begins the auto-clean background worker
func (w *AutoCleanWorker) Start() {
	log.Println("Starting auto-clean worker...")
	w.ticker = time.NewTicker(1 * time.Minute) // Check every minute

	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.checkScheduledCleans()
			case <-w.stopChan:
				w.ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the auto-clean worker
func (w *AutoCleanWorker) Stop() {
	log.Println("Stopping auto-clean worker...")
	w.stopChan <- true
}

// checkScheduledCleans checks for channels that need cleaning
func (w *AutoCleanWorker) checkScheduledCleans() {
	// Don't run if bot is not connected
	if w.bot.Session == nil || w.bot.Session.State == nil || w.bot.Session.State.User == nil {
		log.Printf("[AutoClean] Skipping check - bot not connected")
		return
	}

	rows, err := w.bot.DB.Query(`
		SELECT guild_id, channel_id, interval_hours, warning_minutes, next_run, custom_message, custom_image
		FROM autoclean
		WHERE enabled = 1 AND datetime(next_run) <= datetime('now')
	`)
	if err != nil {
		log.Printf("Error querying autoclean: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var guildID, channelID, nextRunStr, customMessage, customImage string
		var intervalHours, warningMinutes int

		if err := rows.Scan(&guildID, &channelID, &intervalHours, &warningMinutes, &nextRunStr, &customMessage, &customImage); err != nil {
			log.Printf("Error scanning autoclean row: %v", err)
			continue
		}

		// Check if already cleaning this channel (prevent duplicate cleans)
		lockKey := guildID + ":" + channelID
		if _, alreadyCleaning := w.cleaningLocks.LoadOrStore(lockKey, true); alreadyCleaning {
			log.Printf("[AutoClean] Channel %s already being cleaned, skipping", channelID)
			continue
		}

		// Clean the channel
		go w.cleanChannel(guildID, channelID, intervalHours, customMessage, customImage)
	}
}

// cleanChannel performs the actual channel cleaning
func (w *AutoCleanWorker) cleanChannel(guildID, channelID string, intervalHours int, customMessage, customImage string) {
	lockKey := guildID + ":" + channelID

	// Always release the lock when done
	defer w.cleaningLocks.Delete(lockKey)

	log.Printf("[AutoClean] Cleaning channel %s in guild %s", channelID, guildID)

	// Double-check connection before proceeding
	if w.bot.Session == nil || w.bot.Session.State == nil {
		log.Printf("[AutoClean] Aborting clean - bot disconnected")
		w.markCleanFailed(guildID, channelID)
		return
	}

	// Get the original channel
	oldChannel, err := w.bot.Session.Channel(channelID)
	if err != nil {
		log.Printf("[AutoClean] Failed to get channel %s: %v", channelID, err)
		w.markCleanFailed(guildID, channelID)
		return
	}

	// Clone the channel
	newChannel, err := w.bot.Session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:                 oldChannel.Name,
		Type:                 oldChannel.Type,
		Topic:                oldChannel.Topic,
		NSFW:                 oldChannel.NSFW,
		Position:             oldChannel.Position,
		Bitrate:              oldChannel.Bitrate,
		UserLimit:            oldChannel.UserLimit,
		PermissionOverwrites: oldChannel.PermissionOverwrites,
		ParentID:             oldChannel.ParentID,
		RateLimitPerUser:     oldChannel.RateLimitPerUser,
	})

	if err != nil {
		log.Printf("[AutoClean] Failed to clone channel %s: %v", channelID, err)
		w.markCleanFailed(guildID, channelID)
		return
	}

	// CRITICAL: Update database IMMEDIATELY after creating new channel
	// This prevents the loop where old channel keeps getting selected
	nextRun := time.Now().Add(time.Duration(intervalHours) * time.Hour)
	_, err = w.bot.DB.Exec(`
		UPDATE autoclean
		SET channel_id = ?, next_run = ?, last_clean = datetime('now'), warned = 0
		WHERE guild_id = ? AND channel_id = ?`,
		newChannel.ID, nextRun.Format(time.RFC3339), guildID, channelID)

	if err != nil {
		log.Printf("[AutoClean] CRITICAL: Failed to update database with new channel ID: %v", err)
		// Try to delete the new channel since we couldn't update DB
		w.bot.Session.ChannelDelete(newChannel.ID)
		w.markCleanFailed(guildID, channelID)
		return
	}

	log.Printf("[AutoClean] Database updated: old=%s -> new=%s", channelID, newChannel.ID)

	// Move new channel to same position (Discord might not respect position in create)
	newPosition := oldChannel.Position
	_, err = w.bot.Session.ChannelEditComplex(newChannel.ID, &discordgo.ChannelEdit{
		Position: &newPosition,
	})
	if err != nil {
		log.Printf("[AutoClean] Warning: Failed to reposition channel: %v", err)
	}

	// Delete the old channel (non-critical - if this fails, we still have a working new channel)
	_, err = w.bot.Session.ChannelDelete(channelID)
	if err != nil {
		log.Printf("[AutoClean] Warning: Failed to delete old channel %s: %v (new channel %s is active)", channelID, err, newChannel.ID)
		// Don't fail - the new channel is already set up and DB is updated
	}

	// Send completion message
	message := customMessage
	if message == "" {
		message = "🧹 This channel has been automatically cleaned!"
	}

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0xFF51FF,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Auto-cleaned by Yuno",
		},
	}

	if customImage != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: customImage,
		}
	}

	_, err = w.bot.Session.ChannelMessageSendEmbed(newChannel.ID, embed)
	if err != nil {
		log.Printf("[AutoClean] Warning: Failed to send clean message: %v", err)
	}

	log.Printf("[AutoClean] Successfully cleaned channel. Old: %s, New: %s", channelID, newChannel.ID)
}

// markCleanFailed marks a clean as failed and reschedules
func (w *AutoCleanWorker) markCleanFailed(guildID, channelID string) {
	// Retry in 1 hour
	nextRun := time.Now().Add(1 * time.Hour)
	_, err := w.bot.DB.Exec(`
		UPDATE autoclean 
		SET next_run = ? 
		WHERE guild_id = ? AND channel_id = ?`,
		nextRun.Format(time.RFC3339), guildID, channelID)

	if err != nil {
		log.Printf("Failed to reschedule clean: %v", err)
	}
}

// SendWarning sends a warning before cleaning
func (w *AutoCleanWorker) SendWarning(guildID, channelID string, minutesUntilClean int) {
	embed := &discordgo.MessageEmbed{
		Title:       "⚠️ Channel Clean Warning",
		Description: fmt.Sprintf("This channel will be cleaned in **%d minutes**!\n\nAll messages will be deleted.", minutesUntilClean),
		Color:       0xFFAA00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Save any important messages now!",
		},
	}

	_, err := w.bot.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send warning: %v", err)
	}
}

// checkWarnings checks if any channels need warnings
func (w *AutoCleanWorker) checkWarnings() {
	rows, err := w.bot.DB.Query(`
		SELECT guild_id, channel_id, warning_minutes, next_run
		FROM autoclean 
		WHERE enabled = 1 
		AND datetime(next_run, '-' || warning_minutes || ' minutes') <= datetime('now')
		AND datetime(next_run) > datetime('now')
		AND warned = 0
	`)
	if err != nil {
		log.Printf("Error querying warnings: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var guildID, channelID, nextRunStr string
		var warningMinutes int

		if err := rows.Scan(&guildID, &channelID, &warningMinutes, &nextRunStr); err != nil {
			continue
		}

		nextRun, _ := time.Parse(time.RFC3339, nextRunStr)
		minutesUntil := int(time.Until(nextRun).Minutes())

		// Send warning
		w.SendWarning(guildID, channelID, minutesUntil)

		// Mark as warned
		w.bot.DB.Exec(`
			UPDATE autoclean 
			SET warned = 1 
			WHERE guild_id = ? AND channel_id = ?`,
			guildID, channelID)
	}
}
