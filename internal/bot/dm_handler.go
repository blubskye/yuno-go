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
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleDM processes direct messages to the bot
func (b *Bot) handleDM(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer RecoverFromPanic("handleDM")

	user := m.Author

	DebugLog("[DM] Received from %s: %s", user.String(), m.Content)

	// Save to inbox
	attachments := ""
	if len(m.Attachments) > 0 {
		urls := make([]string, len(m.Attachments))
		for i, a := range m.Attachments {
			urls[i] = a.URL
		}
		attachments = strings.Join(urls, ",")
	}

	inboxID, err := b.DB.SaveDM(user.ID, user.String(), m.Content, attachments)
	if err != nil {
		log.Printf("[DM Handler] Failed to save DM: %v", err)
	}

	// Forward to configured channels
	configs, err := b.DB.GetAllDMConfigs()
	if err != nil {
		log.Printf("[DM Handler] Failed to get DM configs: %v", err)
		return
	}

	masterServer := Global.Bot.MasterServer

	for _, config := range configs {
		guildID := config["guild_id"]
		channelID := config["channel_id"]

		guild, err := s.State.Guild(guildID)
		if err != nil {
			continue
		}

		channel, err := s.Channel(channelID)
		if err != nil {
			continue
		}

		isMaster := guildID == masterServer

		// Master server receives ALL DMs, other servers only get member DMs
		if !isMaster {
			// Check if user is a member of this guild
			_, err := s.GuildMember(guildID, user.ID)
			if err != nil {
				continue // User is not a member
			}
		}

		// Create embed
		embed := b.createDMEmbed(user, m.Message, inboxID, isMaster, guild)

		_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
		if err != nil {
			DebugLog("[DM Handler] Failed to forward to %s: %v", channelID, err)
		}
	}

	// Log to terminal
	log.Printf("[DM] %s (%s): %s", user.String(), user.ID, truncateDM(m.Content, 100))
}

// createDMEmbed creates an embed for forwarded DMs
func (b *Bot) createDMEmbed(user *discordgo.User, message *discordgo.Message, inboxID int64, isMaster bool, guild *discordgo.Guild) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    user.String(),
			IconURL: user.AvatarURL("64"),
		},
		Description: message.Content,
		Color:       0x5865F2,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if message.Content == "" {
		embed.Description = "*No text content*"
	}

	// Footer
	footerText := fmt.Sprintf("User ID: %s | Inbox #%d", user.ID, inboxID)
	if isMaster {
		footerText += " | Master Server"
	} else if guild != nil {
		footerText += fmt.Sprintf(" | Member of: %s", guild.Name)
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: footerText}

	// Attachments
	if len(message.Attachments) > 0 {
		var attachmentList []string
		for _, a := range message.Attachments {
			attachmentList = append(attachmentList, fmt.Sprintf("[%s](%s)", a.Filename, a.URL))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Attachments",
			Value: strings.Join(attachmentList, "\n"),
		})

		// Set first image as thumbnail
		for _, a := range message.Attachments {
			if strings.HasPrefix(a.ContentType, "image/") {
				embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: a.URL}
				break
			}
		}
	}

	return embed
}

// truncateDM truncates a DM for logging
func truncateDM(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// StartDMCleanup starts the periodic DM cleanup task
func (b *Bot) StartDMCleanup() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		deleted, err := b.DB.ClearOldDMs(30) // Keep 30 days
		if err == nil && deleted > 0 {
			log.Printf("[DM Handler] Cleaned up %d old DM(s)", deleted)
		}
	}
}
