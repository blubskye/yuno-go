/*
    Yuno Gasai. A Discord.JS based bot, with multiple features.
    Copyright (C) 2018 Maeeen <maeeennn@gmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see https://www.gnu.org/licenses/.
*/

package bot

import (
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// VoiceXPTracker tracks voice channel sessions for XP
type VoiceXPTracker struct {
	bot        *Bot
	sessions   map[string]map[string]*VoiceSession // guildID -> userID -> session
	mu         sync.RWMutex
	ticker     *time.Ticker
	stopChan   chan struct{}
	running    bool
}

// VoiceSession represents an active voice session
type VoiceSession struct {
	GuildID   string
	UserID    string
	ChannelID string
	JoinedAt  time.Time
	LastXP    time.Time
}

// NewVoiceXPTracker creates a new voice XP tracker
func NewVoiceXPTracker(b *Bot) *VoiceXPTracker {
	return &VoiceXPTracker{
		bot:      b,
		sessions: make(map[string]map[string]*VoiceSession),
		stopChan: make(chan struct{}),
	}
}

// Start begins the voice XP tracking
func (v *VoiceXPTracker) Start() {
	v.running = true

	// Recover existing sessions from database
	v.recoverSessions()

	// Start the XP ticker (check every minute)
	v.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-v.ticker.C:
				v.processXP()
			case <-v.stopChan:
				return
			}
		}
	}()

	DebugLog("Voice XP tracker started")
}

// Stop stops the voice XP tracker
func (v *VoiceXPTracker) Stop() {
	v.running = false
	if v.ticker != nil {
		v.ticker.Stop()
	}
	close(v.stopChan)

	// Save all sessions to database for recovery
	v.mu.RLock()
	for _, guildSessions := range v.sessions {
		for _, session := range guildSessions {
			v.bot.DB.SaveVoiceSession(session.GuildID, session.UserID, session.ChannelID)
		}
	}
	v.mu.RUnlock()
}

// recoverSessions recovers voice sessions from database after restart
func (v *VoiceXPTracker) recoverSessions() {
	sessions, err := v.bot.DB.GetAllVoiceSessions()
	if err != nil {
		log.Printf("[Voice XP] Error recovering sessions: %v", err)
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	recovered := 0
	for _, s := range sessions {
		guildID := s["guild_id"]
		userID := s["user_id"]
		channelID := s["channel_id"]

		// Verify user is still in voice
		vs, err := v.bot.Session.State.VoiceState(guildID, userID)
		if err != nil || vs.ChannelID == "" {
			// User no longer in voice, clean up
			v.bot.DB.RemoveVoiceSession(guildID, userID)
			continue
		}

		if v.sessions[guildID] == nil {
			v.sessions[guildID] = make(map[string]*VoiceSession)
		}

		v.sessions[guildID][userID] = &VoiceSession{
			GuildID:   guildID,
			UserID:    userID,
			ChannelID: channelID,
			JoinedAt:  time.Now(), // Reset timer on recovery
			LastXP:    time.Now(),
		}
		recovered++
	}

	if recovered > 0 {
		log.Printf("[Voice XP] Recovered %d voice sessions", recovered)
	}
}

// HandleVoiceStateUpdate processes voice state changes for XP tracking
func (v *VoiceXPTracker) HandleVoiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	if !v.running {
		return
	}

	// Ignore bots
	if vs.Member != nil && vs.Member.User.Bot {
		return
	}

	guildID := vs.GuildID
	userID := vs.UserID

	// Check if voice XP is enabled for this guild
	enabled, _, _, _, err := v.bot.DB.GetVoiceXPConfig(guildID)
	if err != nil || !enabled {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// Initialize guild map if needed
	if v.sessions[guildID] == nil {
		v.sessions[guildID] = make(map[string]*VoiceSession)
	}

	// User left voice entirely
	if vs.ChannelID == "" {
		if session, ok := v.sessions[guildID][userID]; ok {
			DebugLog("[Voice XP] User %s left voice in guild %s", userID, guildID)
			delete(v.sessions[guildID], userID)
			v.bot.DB.RemoveVoiceSession(session.GuildID, session.UserID)
		}
		return
	}

	// User joined or moved channels
	if _, ok := v.sessions[guildID][userID]; !ok {
		// New session
		DebugLog("[Voice XP] User %s joined voice in guild %s", userID, guildID)
		v.sessions[guildID][userID] = &VoiceSession{
			GuildID:   guildID,
			UserID:    userID,
			ChannelID: vs.ChannelID,
			JoinedAt:  time.Now(),
			LastXP:    time.Now(),
		}
		v.bot.DB.SaveVoiceSession(guildID, userID, vs.ChannelID)
	} else {
		// Channel change
		v.sessions[guildID][userID].ChannelID = vs.ChannelID
	}
}

// processXP grants XP to users in voice channels
func (v *VoiceXPTracker) processXP() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	now := time.Now()

	for guildID, guildSessions := range v.sessions {
		// Get guild config
		enabled, xpRate, intervalSec, ignoreAFK, err := v.bot.DB.GetVoiceXPConfig(guildID)
		if err != nil || !enabled {
			continue
		}

		interval := time.Duration(intervalSec) * time.Second

		// Get AFK channel ID if we need to ignore it
		var afkChannelID string
		if ignoreAFK {
			guild, err := v.bot.Session.State.Guild(guildID)
			if err == nil {
				afkChannelID = guild.AfkChannelID
			}
		}

		for userID, session := range guildSessions {
			// Check if enough time has passed since last XP
			if now.Sub(session.LastXP) < interval {
				continue
			}

			// Skip AFK channel if configured
			if ignoreAFK && session.ChannelID == afkChannelID {
				continue
			}

			// Grant XP
			go v.bot.giveXP(v.bot.Session, guildID, userID, "", xpRate)
			session.LastXP = now

			DebugLog("[Voice XP] Granted %d XP to %s in guild %s", xpRate, userID, guildID)
		}
	}
}

// GetActiveSessions returns count of active voice sessions
func (v *VoiceXPTracker) GetActiveSessions(guildID string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if guildID == "" {
		total := 0
		for _, gs := range v.sessions {
			total += len(gs)
		}
		return total
	}

	return len(v.sessions[guildID])
}

// GetSessionsForGuild returns active sessions for a guild
func (v *VoiceXPTracker) GetSessionsForGuild(guildID string) []*VoiceSession {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var sessions []*VoiceSession
	if guildSessions, ok := v.sessions[guildID]; ok {
		for _, s := range guildSessions {
			sessions = append(sessions, s)
		}
	}
	return sessions
}
