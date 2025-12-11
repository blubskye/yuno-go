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
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SetVCXPCommand configures voice channel XP
type SetVCXPCommand struct{}

func (c *SetVCXPCommand) Name() string        { return "set-vcxp" }
func (c *SetVCXPCommand) Aliases() []string   { return []string{"setvcxp", "vcxp-config", "voice-xp"} }
func (c *SetVCXPCommand) Description() string { return "Configure voice channel XP settings" }
func (c *SetVCXPCommand) Usage() string {
	return "set-vcxp <enable|disable|rate|interval|ignore-afk> [value]"
}
func (c *SetVCXPCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetVCXPCommand) MasterOnly() bool { return false }

func (c *SetVCXPCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"⚙️ **Voice XP Configuration**\n\n"+
				"**Usage:**\n"+
				"`set-vcxp enable` - Enable voice XP\n"+
				"`set-vcxp disable` - Disable voice XP\n"+
				"`set-vcxp rate <amount>` - Set XP per interval (default: 10)\n"+
				"`set-vcxp interval <seconds>` - Set interval (default: 300)\n"+
				"`set-vcxp ignore-afk <true|false>` - Ignore AFK channel\n\n"+
				"**Examples:**\n"+
				"`set-vcxp enable`\n"+
				"`set-vcxp rate 15`\n"+
				"`set-vcxp interval 300`")
		return nil
	}

	bot := ctx.Bot.(interface {
		GetDB() interface {
			GetVoiceXPConfig(guildID string) (bool, int, int, bool, error)
			SetVoiceXPConfig(guildID string, enabled bool, xpRate, interval int, ignoreAFK bool) error
		}
	})

	// Get current config
	enabled, xpRate, interval, ignoreAFK, _ := bot.GetDB().GetVoiceXPConfig(ctx.Message.GuildID)

	subcommand := strings.ToLower(ctx.Args[0])

	switch subcommand {
	case "enable", "on":
		enabled = true
		err := bot.GetDB().SetVoiceXPConfig(ctx.Message.GuildID, enabled, xpRate, interval, ignoreAFK)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("❌ Error: %v", err))
			return err
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "✅ Voice XP has been **enabled**!")

	case "disable", "off":
		enabled = false
		err := bot.GetDB().SetVoiceXPConfig(ctx.Message.GuildID, enabled, xpRate, interval, ignoreAFK)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("❌ Error: %v", err))
			return err
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "✅ Voice XP has been **disabled**.")

	case "rate", "xp":
		if len(ctx.Args) < 2 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Please provide XP amount. Example: `set-vcxp rate 15`")
			return nil
		}
		newRate, err := strconv.Atoi(ctx.Args[1])
		if err != nil || newRate < 1 || newRate > 1000 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Rate must be between 1 and 1000.")
			return nil
		}
		xpRate = newRate
		err = bot.GetDB().SetVoiceXPConfig(ctx.Message.GuildID, enabled, xpRate, interval, ignoreAFK)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("❌ Error: %v", err))
			return err
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("✅ Voice XP rate set to **%d** per interval.", xpRate))

	case "interval", "time":
		if len(ctx.Args) < 2 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Please provide interval in seconds. Example: `set-vcxp interval 300`")
			return nil
		}
		newInterval, err := strconv.Atoi(ctx.Args[1])
		if err != nil || newInterval < 60 || newInterval > 3600 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Interval must be between 60 and 3600 seconds.")
			return nil
		}
		interval = newInterval
		err = bot.GetDB().SetVoiceXPConfig(ctx.Message.GuildID, enabled, xpRate, interval, ignoreAFK)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("❌ Error: %v", err))
			return err
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("✅ Voice XP interval set to **%d** seconds (%d min).", interval, interval/60))

	case "ignore-afk", "ignoreafk", "afk":
		if len(ctx.Args) < 2 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Please provide true or false. Example: `set-vcxp ignore-afk true`")
			return nil
		}
		val := strings.ToLower(ctx.Args[1])
		if val == "true" || val == "yes" || val == "1" {
			ignoreAFK = true
		} else if val == "false" || val == "no" || val == "0" {
			ignoreAFK = false
		} else {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Please provide true or false.")
			return nil
		}
		err := bot.GetDB().SetVoiceXPConfig(ctx.Message.GuildID, enabled, xpRate, interval, ignoreAFK)
		if err != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("❌ Error: %v", err))
			return err
		}
		if ignoreAFK {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "✅ AFK channel will be **ignored** for voice XP.")
		} else {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "✅ AFK channel will **earn** voice XP.")
		}

	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Unknown option. Use `enable`, `disable`, `rate`, `interval`, or `ignore-afk`.")
	}

	return nil
}

// VCXPStatusCommand shows voice XP status
type VCXPStatusCommand struct{}

func (c *VCXPStatusCommand) Name() string        { return "vcxp-status" }
func (c *VCXPStatusCommand) Aliases() []string   { return []string{"vcxpstatus", "voice-status"} }
func (c *VCXPStatusCommand) Description() string { return "View voice XP configuration and active sessions" }
func (c *VCXPStatusCommand) Usage() string       { return "vcxp-status" }
func (c *VCXPStatusCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageGuild}
}
func (c *VCXPStatusCommand) MasterOnly() bool { return false }

func (c *VCXPStatusCommand) Execute(ctx *Context) error {
	bot := ctx.Bot.(interface {
		GetDB() interface {
			GetVoiceXPConfig(guildID string) (bool, int, int, bool, error)
		}
		GetVoiceXPTracker() interface {
			GetActiveSessions(guildID string) int
			GetSessionsForGuild(guildID string) interface{}
		}
	})

	enabled, xpRate, interval, ignoreAFK, _ := bot.GetDB().GetVoiceXPConfig(ctx.Message.GuildID)
	activeSessions := bot.GetVoiceXPTracker().GetActiveSessions(ctx.Message.GuildID)

	statusEmoji := "❌"
	statusText := "Disabled"
	if enabled {
		statusEmoji = "✅"
		statusText = "Enabled"
	}

	afkText := "No"
	if ignoreAFK {
		afkText = "Yes"
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎤 Voice XP Status",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  fmt.Sprintf("%s %s", statusEmoji, statusText),
				Inline: true,
			},
			{
				Name:   "XP Rate",
				Value:  fmt.Sprintf("%d per interval", xpRate),
				Inline: true,
			},
			{
				Name:   "Interval",
				Value:  fmt.Sprintf("%d seconds (%d min)", interval, interval/60),
				Inline: true,
			},
			{
				Name:   "Ignore AFK",
				Value:  afkText,
				Inline: true,
			},
			{
				Name:   "Active Sessions",
				Value:  fmt.Sprintf("%d user(s) in voice", activeSessions),
				Inline: true,
			},
		},
	}

	if !enabled {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Use 'set-vcxp enable' to enable voice XP",
		}
	}

	ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return nil
}
