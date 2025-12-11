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

package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// BotBanCommand bans a user or server from using the bot
type BotBanCommand struct{}

func (c *BotBanCommand) Name() string        { return "bot-ban" }
func (c *BotBanCommand) Aliases() []string   { return []string{"botban"} }
func (c *BotBanCommand) Description() string { return "Ban a user or server from using the bot" }
func (c *BotBanCommand) Usage() string {
	return "bot-ban <user|server> <id> [reason]"
}
func (c *BotBanCommand) RequiredPermissions() []int64 { return nil }
func (c *BotBanCommand) MasterOnly() bool             { return true }

func (c *BotBanCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ **Usage:** `bot-ban <user|server> <id> [reason]`\n\n"+
				"**Examples:**\n"+
				"• `bot-ban user 123456789 Spamming`\n"+
				"• `bot-ban server 987654321 Abuse`")
		return nil
	}

	banType := strings.ToLower(ctx.Args[0])
	if banType != "user" && banType != "server" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Invalid type. Use `user` or `server`.")
		return nil
	}

	targetID := ctx.Args[1]
	reason := "No reason provided"
	if len(ctx.Args) > 2 {
		reason = strings.Join(ctx.Args[2:], " ")
	}

	// Get database from bot
	bot := ctx.Bot.(interface {
		GetDB() interface {
			AddBotBan(id, banType, reason, bannedBy string) error
		}
	})

	err := bot.GetDB().AddBotBan(targetID, banType, reason, ctx.Message.Author.ID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to add ban: %v", err))
		return err
	}

	emoji := "👤"
	if banType == "server" {
		emoji = "🏠"
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("%s **Bot-banned %s** `%s`\n**Reason:** %s",
			emoji, banType, targetID, reason))
	return nil
}

// BotUnbanCommand removes a bot-level ban
type BotUnbanCommand struct{}

func (c *BotUnbanCommand) Name() string        { return "bot-unban" }
func (c *BotUnbanCommand) Aliases() []string   { return []string{"botunban"} }
func (c *BotUnbanCommand) Description() string { return "Remove a bot-level ban from a user or server" }
func (c *BotUnbanCommand) Usage() string       { return "bot-unban <id>" }
func (c *BotUnbanCommand) RequiredPermissions() []int64 { return nil }
func (c *BotUnbanCommand) MasterOnly() bool             { return true }

func (c *BotUnbanCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ **Usage:** `bot-unban <id>`")
		return nil
	}

	targetID := ctx.Args[0]

	bot := ctx.Bot.(interface {
		GetDB() interface {
			RemoveBotBan(id string) error
		}
	})

	err := bot.GetDB().RemoveBotBan(targetID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to remove ban: %v", err))
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("✅ **Removed bot-ban** for `%s`", targetID))
	return nil
}

// BotBanlistCommand lists all bot-level bans
type BotBanlistCommand struct{}

func (c *BotBanlistCommand) Name() string        { return "bot-banlist" }
func (c *BotBanlistCommand) Aliases() []string   { return []string{"botbanlist", "bot-bans"} }
func (c *BotBanlistCommand) Description() string { return "List all bot-level bans" }
func (c *BotBanlistCommand) Usage() string       { return "bot-banlist [users|servers]" }
func (c *BotBanlistCommand) RequiredPermissions() []int64 { return nil }
func (c *BotBanlistCommand) MasterOnly() bool             { return true }

func (c *BotBanlistCommand) Execute(ctx *Context) error {
	filterType := ""
	if len(ctx.Args) > 0 {
		switch strings.ToLower(ctx.Args[0]) {
		case "users", "user":
			filterType = "user"
		case "servers", "server":
			filterType = "server"
		}
	}

	bot := ctx.Bot.(interface {
		GetDB() interface {
			GetBotBans(banType string) ([]map[string]string, error)
		}
	})

	bans, err := bot.GetDB().GetBotBans(filterType)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to get bans: %v", err))
		return err
	}

	if len(bans) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"📋 No bot-level bans found.")
		return nil
	}

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title: "🚫 Bot-Level Bans",
		Color: 0xFF0000,
	}

	var userBans, serverBans []string
	for _, ban := range bans {
		entry := fmt.Sprintf("`%s` - %s", ban["id"], ban["reason"])
		if ban["type"] == "user" {
			userBans = append(userBans, entry)
		} else {
			serverBans = append(serverBans, entry)
		}
	}

	if len(userBans) > 0 && (filterType == "" || filterType == "user") {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("👤 Users (%d)", len(userBans)),
			Value: truncateString(strings.Join(userBans, "\n"), 1024),
		})
	}

	if len(serverBans) > 0 && (filterType == "" || filterType == "server") {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("🏠 Servers (%d)", len(serverBans)),
			Value: truncateString(strings.Join(serverBans, "\n"), 1024),
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Total: %d ban(s)", len(bans)),
	}

	ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return nil
}

// Helper to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
