package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// onMessageDelete handles message deletion logging
func (b *Bot) onMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	if m.GuildID == "" {
		return
	}

	// Use cached config for high-frequency events
	config, err := b.GetLoggingConfigCached(m.GuildID)
	if err != nil || !config.Enabled || !config.MessageDelete || config.LogChannelID == "" {
		return
	}

	// Check channel override
	enabled, _ := b.IsChannelLoggingEnabled(m.GuildID, m.ChannelID, "message_delete")
	if !enabled {
		return
	}

	// Try to get cached message
	cached, err := b.GetCachedMessage(m.ID)

	embed := &discordgo.MessageEmbed{
		Title: "Message Deleted",
		Color: 0xe74c3c, // Red
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", m.ChannelID),
				Inline: true,
			},
			{
				Name:   "Message ID",
				Value:  m.ID,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if cached != nil && cached.Author != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Author",
			Value:  fmt.Sprintf("<@%s>", cached.Author.ID),
			Inline: true,
		})

		if cached.Content != "" {
			content := cached.Content
			if len(content) > 1024 {
				content = content[:1021] + "..."
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Content",
				Value:  content,
				Inline: false,
			})
		}

		if len(m.Attachments) > 0 {
			attachmentURLs := ""
			for _, att := range m.Attachments {
				attachmentURLs += att.URL + "\n"
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Attachments",
				Value:  attachmentURLs,
				Inline: false,
			})
		}
	}

	_, err = s.ChannelMessageSendEmbed(config.LogChannelID, embed)
	if err != nil {
		log.Printf("Failed to log message deletion: %v", err)
	}
}

// onMessageUpdate handles message edit logging
func (b *Bot) onMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if m.GuildID == "" || m.Author == nil || m.Author.Bot {
		return
	}

	// Use cached config for high-frequency events
	config, err := b.GetLoggingConfigCached(m.GuildID)
	if err != nil || !config.Enabled || !config.MessageEdit || config.LogChannelID == "" {
		return
	}

	// Check channel override
	enabled, _ := b.IsChannelLoggingEnabled(m.GuildID, m.ChannelID, "message_edit")
	if !enabled {
		return
	}

	// Get the old message from cache
	cached, err := b.GetCachedMessage(m.ID)
	if err != nil || cached == nil {
		return
	}

	// If content is the same, it's not an edit (probably embed update)
	if cached.Content == m.Content {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "Message Edited",
		Color: 0xf39c12, // Orange
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Author",
				Value:  fmt.Sprintf("<@%s>", m.Author.ID),
				Inline: true,
			},
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", m.ChannelID),
				Inline: true,
			},
			{
				Name:   "Message",
				Value:  fmt.Sprintf("[Jump to Message](https://discord.com/channels/%s/%s/%s)", m.GuildID, m.ChannelID, m.ID),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	oldContent := cached.Content
	if len(oldContent) > 1024 {
		oldContent = oldContent[:1021] + "..."
	}
	newContent := m.Content
	if len(newContent) > 1024 {
		newContent = newContent[:1021] + "..."
	}

	if oldContent != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Before",
			Value:  oldContent,
			Inline: false,
		})
	}

	if newContent != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "After",
			Value:  newContent,
			Inline: false,
		})
	}

	_, err = s.ChannelMessageSendEmbed(config.LogChannelID, embed)
	if err != nil {
		log.Printf("Failed to log message edit: %v", err)
	}

	// Update cache with new content
	b.CacheMessage(m.Message)
}

// onVoiceStateUpdateLogging handles voice channel join/leave logging (batched)
func (b *Bot) onVoiceStateUpdateLogging(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	if v.GuildID == "" {
		return
	}

	// Use cached config for high-frequency events
	config, err := b.GetLoggingConfigCached(v.GuildID)
	if err != nil || !config.Enabled || config.LogChannelID == "" {
		return
	}

	// Determine if user joined or left and add to batch
	if v.BeforeUpdate != nil {
		// User was in a voice channel before
		if v.ChannelID == "" {
			// User left voice channel
			if config.MemberLeaveVoice {
				b.addVoiceEventToBatch(v.GuildID, v.UserID, v.BeforeUpdate.ChannelID, "voice_leave")
			}
		} else if v.BeforeUpdate.ChannelID != v.ChannelID {
			// User switched voice channels - log both
			if config.MemberLeaveVoice {
				b.addVoiceEventToBatch(v.GuildID, v.UserID, v.BeforeUpdate.ChannelID, "voice_leave")
			}
			if config.MemberJoinVoice {
				b.addVoiceEventToBatch(v.GuildID, v.UserID, v.ChannelID, "voice_join")
			}
		}
		// Just a state update (mute/deafen), don't log
	} else {
		// User joined voice channel
		if config.MemberJoinVoice {
			b.addVoiceEventToBatch(v.GuildID, v.UserID, v.ChannelID, "voice_join")
		}
	}
}

// addVoiceEventToBatch adds a voice event to the event log batcher
func (b *Bot) addVoiceEventToBatch(guildID, userID, channelID, eventType string) {
	if b.EventLogBatcher == nil {
		return
	}

	b.EventLogBatcher.AddEvent(guildID, LogEvent{
		Type:      eventType,
		UserID:    userID,
		ChannelID: channelID,
		Timestamp: time.Now(),
	})
}

// onGuildMemberUpdate handles nickname and avatar changes (batched)
func (b *Bot) onGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	// Use cached config for high-frequency events
	config, err := b.GetLoggingConfigCached(m.GuildID)
	if err != nil || !config.Enabled || config.LogChannelID == "" {
		return
	}

	// Check for nickname change
	if config.NicknameChange && m.BeforeUpdate != nil {
		oldNick := m.BeforeUpdate.Nick
		newNick := m.Nick

		if oldNick != newNick && b.EventLogBatcher != nil {
			b.EventLogBatcher.AddEvent(m.GuildID, LogEvent{
				Type:      "nickname",
				UserID:    m.User.ID,
				Username:  m.User.Username,
				OldValue:  oldNick,
				NewValue:  newNick,
				Timestamp: time.Now(),
			})
		}
	}

	// Check for avatar change
	if config.AvatarChange && m.BeforeUpdate != nil && m.User != nil {
		oldAvatar := m.BeforeUpdate.Avatar
		newAvatar := m.User.Avatar

		if oldAvatar != newAvatar && b.EventLogBatcher != nil {
			extra := make(map[string]string)
			if newAvatar != "" {
				extra["new_avatar_url"] = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", m.User.ID, newAvatar)
			}
			if oldAvatar != "" {
				extra["old_avatar_url"] = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", m.User.ID, oldAvatar)
			}

			b.EventLogBatcher.AddEvent(m.GuildID, LogEvent{
				Type:      "avatar",
				UserID:    m.User.ID,
				Username:  m.User.Username,
				OldValue:  oldAvatar,
				NewValue:  newAvatar,
				Timestamp: time.Now(),
				Extra:     extra,
			})
		}
	}
}

// onPresenceUpdate handles presence changes (online/offline/etc)
func (b *Bot) onPresenceUpdate(s *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p.GuildID == "" {
		return
	}

	// Use cached config for very high-frequency events
	config, err := b.GetLoggingConfigCached(p.GuildID)
	if err != nil || !config.Enabled || !config.PresenceChange || config.LogChannelID == "" {
		return
	}

	// Get old and new status
	oldStatus := "unknown"
	if p.Presence.Status != "" {
		oldStatus = string(p.Presence.Status)
	}

	newStatus := "unknown"
	if p.Status != "" {
		newStatus = string(p.Status)
	}

	// Don't log if status hasn't changed
	if oldStatus == newStatus {
		return
	}

	// Add to batch
	username := "Unknown User"
	if p.User != nil {
		username = p.User.Username
	}

	b.PresenceBatcher.AddChange(p.GuildID, PresenceChange{
		UserID:    p.User.ID,
		Username:  username,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
	})
}
