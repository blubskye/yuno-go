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
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SetDMChannelCommand sets the DM forwarding channel
type SetDMChannelCommand struct{}

func (c *SetDMChannelCommand) Name() string        { return "set-dm-channel" }
func (c *SetDMChannelCommand) Aliases() []string   { return []string{"setdmchannel", "dm-channel"} }
func (c *SetDMChannelCommand) Description() string { return "Set or remove the DM forwarding channel" }
func (c *SetDMChannelCommand) Usage() string       { return "set-dm-channel <#channel|none>" }
func (c *SetDMChannelCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetDMChannelCommand) MasterOnly() bool { return false }

func (c *SetDMChannelCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ **Usage:** `set-dm-channel <#channel|none>`\n\n"+
				"**Examples:**\n"+
				"• `set-dm-channel #bot-dms` - Set DM forwarding channel\n"+
				"• `set-dm-channel none` - Disable DM forwarding")
		return nil
	}

	bot := ctx.Bot.(interface {
		GetDB() interface {
			SetDMConfig(guildID, channelID string) error
			RemoveDMConfig(guildID string) error
		}
	})

	arg := strings.ToLower(ctx.Args[0])

	// Handle removal
	if arg == "none" || arg == "disable" || arg == "off" {
		err := bot.GetDB().RemoveDMConfig(ctx.Message.GuildID)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				fmt.Sprintf("❌ Failed to remove config: %v", err))
			return err
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"✅ DM forwarding has been **disabled** for this server.")
		return nil
	}

	// Parse channel mention or ID
	channelID := arg
	channelMention := regexp.MustCompile(`<#(\d+)>`)
	if matches := channelMention.FindStringSubmatch(ctx.Args[0]); len(matches) > 1 {
		channelID = matches[1]
	}

	// Verify channel exists
	channel, err := ctx.Session.Channel(channelID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Could not find that channel. Make sure it's a valid text channel.")
		return nil
	}

	// Verify it's a text channel in this guild
	if channel.GuildID != ctx.Message.GuildID {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ That channel is not in this server.")
		return nil
	}

	err = bot.GetDB().SetDMConfig(ctx.Message.GuildID, channelID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to set config: %v", err))
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("✅ DM forwarding channel set to <#%s>", channelID))
	return nil
}

// DMStatusCommand shows DM forwarding status
type DMStatusCommand struct{}

func (c *DMStatusCommand) Name() string        { return "dm-status" }
func (c *DMStatusCommand) Aliases() []string   { return []string{"dmstatus", "dm-config"} }
func (c *DMStatusCommand) Description() string { return "View DM forwarding configuration for this server" }
func (c *DMStatusCommand) Usage() string       { return "dm-status" }
func (c *DMStatusCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageGuild}
}
func (c *DMStatusCommand) MasterOnly() bool { return false }

func (c *DMStatusCommand) Execute(ctx *Context) error {
	bot := ctx.Bot.(interface {
		GetDB() interface {
			GetDMConfig(guildID string) (string, bool, error)
			GetUnreadDMCount() (int, error)
		}
		GetMasterServer() string
	})

	channelID, enabled, err := bot.GetDB().GetDMConfig(ctx.Message.GuildID)
	masterServer := bot.GetMasterServer()
	isMaster := ctx.Message.GuildID == masterServer
	unreadCount, _ := bot.GetDB().GetUnreadDMCount()

	embed := &discordgo.MessageEmbed{
		Title:     "📬 DM Forwarding Status",
		Timestamp: "",
	}

	if err != nil || channelID == "" {
		embed.Color = 0xFF0000
		embed.Description = "❌ DM forwarding is **not configured** for this server.\n\n" +
			"Use `set-dm-channel #channel` to enable."
	} else {
		embed.Color = 0x00FF00

		statusText := "Active"
		if !enabled {
			statusText = "Disabled"
		}

		description := "✅ DM forwarding is **enabled**.\n\n"
		description += fmt.Sprintf("**Channel:** <#%s>\n", channelID)
		description += fmt.Sprintf("**Status:** %s\n", statusText)

		if isMaster {
			description += "\n⭐ **Master Server** - All DMs are forwarded here."
		} else {
			description += "\nOnly DMs from this server's members are forwarded."
		}

		embed.Description = description
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Inbox Status",
			Value:  fmt.Sprintf("**%d** unread DM(s) in the inbox.\n\nUse the terminal `inbox` command to view.", unreadCount),
			Inline: false,
		},
	}

	if isMaster {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Master Server"}
	}

	ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return nil
}
