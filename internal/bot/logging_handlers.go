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

	config, err := b.GetLoggingConfig(m.GuildID)
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

	config, err := b.GetLoggingConfig(m.GuildID)
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

// onVoiceStateUpdate handles voice channel join/leave logging
func (b *Bot) onVoiceStateUpdateLogging(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	if v.GuildID == "" {
		return
	}

	config, err := b.GetLoggingConfig(v.GuildID)
	if err != nil || !config.Enabled || config.LogChannelID == "" {
		return
	}

	// Determine if user joined or left
	var logType string
	var title string
	var color int
	var channelID string

	if v.BeforeUpdate != nil {
		// User was in a voice channel before
		if v.ChannelID == "" {
			// User left voice channel
			logType = "member_leave_voice"
			title = "Member Left Voice Channel"
			color = 0xe74c3c // Red
			channelID = v.BeforeUpdate.ChannelID
		} else if v.BeforeUpdate.ChannelID != v.ChannelID {
			// User switched voice channels - log both
			b.logVoiceEvent(s, config, v, "member_leave_voice", "Member Left Voice Channel", 0xe74c3c, v.BeforeUpdate.ChannelID)
			logType = "member_join_voice"
			title = "Member Joined Voice Channel"
			color = 0x2ecc71 // Green
			channelID = v.ChannelID
		} else {
			// Just a state update (mute/deafen), don't log
			return
		}
	} else {
		// User joined voice channel
		logType = "member_join_voice"
		title = "Member Joined Voice Channel"
		color = 0x2ecc71 // Green
		channelID = v.ChannelID
	}

	b.logVoiceEvent(s, config, v, logType, title, color, channelID)
}

// logVoiceEvent is a helper function to log voice events
func (b *Bot) logVoiceEvent(s *discordgo.Session, config *LoggingConfig, v *discordgo.VoiceStateUpdate, logType, title string, color int, channelID string) {
	// Check if this type is enabled
	enabled := false
	switch logType {
	case "member_join_voice":
		enabled = config.MemberJoinVoice
	case "member_leave_voice":
		enabled = config.MemberLeaveVoice
	}

	if !enabled {
		return
	}

	// Check channel override (voice channel)
	if channelID != "" {
		overrideEnabled, _ := b.IsChannelLoggingEnabled(v.GuildID, channelID, logType)
		if !overrideEnabled {
			return
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
		Color: color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Member",
				Value:  fmt.Sprintf("<@%s>", v.UserID),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if channelID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Channel",
			Value:  fmt.Sprintf("<#%s>", channelID),
			Inline: true,
		})
	}

	_, err := s.ChannelMessageSendEmbed(config.LogChannelID, embed)
	if err != nil {
		log.Printf("Failed to log voice state update: %v", err)
	}
}

// onGuildMemberUpdate handles nickname and avatar changes
func (b *Bot) onGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	config, err := b.GetLoggingConfig(m.GuildID)
	if err != nil || !config.Enabled || config.LogChannelID == "" {
		return
	}

	// Check for nickname change
	if config.NicknameChange && m.BeforeUpdate != nil {
		oldNick := m.BeforeUpdate.Nick
		newNick := m.Nick

		if oldNick != newNick {
			embed := &discordgo.MessageEmbed{
				Title: "Nickname Changed",
				Color: 0x9b59b6, // Purple
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Member",
						Value:  fmt.Sprintf("<@%s>", m.User.ID),
						Inline: true,
					},
					{
						Name:   "Before",
						Value:  oldNick,
						Inline: true,
					},
					{
						Name:   "After",
						Value:  newNick,
						Inline: true,
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			if oldNick == "" {
				embed.Fields[1].Value = m.User.Username
			}
			if newNick == "" {
				embed.Fields[2].Value = m.User.Username
			}

			_, err = s.ChannelMessageSendEmbed(config.LogChannelID, embed)
			if err != nil {
				log.Printf("Failed to log nickname change: %v", err)
			}
		}
	}

	// Check for avatar change
	if config.AvatarChange && m.BeforeUpdate != nil && m.User != nil {
		oldAvatar := m.BeforeUpdate.Avatar
		newAvatar := m.User.Avatar

		if oldAvatar != newAvatar {
			embed := &discordgo.MessageEmbed{
				Title: "Avatar Changed",
				Color: 0x1abc9c, // Turquoise
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Member",
						Value:  fmt.Sprintf("<@%s>", m.User.ID),
						Inline: true,
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			if oldAvatar != "" {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "Old Avatar",
					Value:  fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", m.User.ID, oldAvatar),
					Inline: false,
				})
			}

			if newAvatar != "" {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "New Avatar",
					Value:  fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", m.User.ID, newAvatar),
					Inline: false,
				})
				embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
					URL: fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", m.User.ID, newAvatar),
				}
			}

			_, err = s.ChannelMessageSendEmbed(config.LogChannelID, embed)
			if err != nil {
				log.Printf("Failed to log avatar change: %v", err)
			}
		}
	}
}

// onPresenceUpdate handles presence changes (online/offline/etc)
func (b *Bot) onPresenceUpdate(s *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p.GuildID == "" {
		return
	}

	config, err := b.GetLoggingConfig(p.GuildID)
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
