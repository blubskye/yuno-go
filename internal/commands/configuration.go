package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SetPrefixCommand changes the bot prefix for a guild
type SetPrefixCommand struct {
	DB interface {
		SetGuildPrefix(guildID, prefix string) error
		GetGuildPrefix(guildID string) (string, error)
	}
}

func (c *SetPrefixCommand) Name() string        { return "set-prefix" }
func (c *SetPrefixCommand) Aliases() []string   { return []string{"setprefix", "prefix"} }
func (c *SetPrefixCommand) Description() string { return "Set the command prefix for this server" }
func (c *SetPrefixCommand) Usage() string       { return "set-prefix <new prefix>" }
func (c *SetPrefixCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetPrefixCommand) MasterOnly() bool { return false }

func (c *SetPrefixCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) == 0 {
		// Show current prefix
		prefix, err := c.DB.GetGuildPrefix(ctx.Message.GuildID)
		if err != nil {
			prefix = ctx.GetPrefix()
		}
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("Current prefix: `%s`\nUsage: `%sset-prefix <new prefix>`", prefix, prefix))
		return nil
	}

	newPrefix := ctx.Args[0]
	if len(newPrefix) > 10 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Prefix must be 10 characters or less.")
		return nil
	}

	if err := c.DB.SetGuildPrefix(ctx.Message.GuildID, newPrefix); err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error saving prefix. Please try again.")
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Prefix Updated",
		Description: fmt.Sprintf("Command prefix set to `%s`", newPrefix),
		Color:       0x00FF00,
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetPresenceCommand changes the bot's presence/status
type SetPresenceCommand struct{}

func (c *SetPresenceCommand) Name() string      { return "set-presence" }
func (c *SetPresenceCommand) Aliases() []string { return []string{"setpresence", "setstatus", "presence", "activity"} }
func (c *SetPresenceCommand) Description() string {
	return "Set the bot's activity status and online presence"
}
func (c *SetPresenceCommand) Usage() string {
	return "set-presence <type> <text> | set-presence status <status> | set-presence clear"
}
func (c *SetPresenceCommand) RequiredPermissions() []int64 { return nil }
func (c *SetPresenceCommand) MasterOnly() bool             { return true }

func (c *SetPresenceCommand) Execute(ctx *Context) error {
	if len(ctx.Args) < 1 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"ℹ️ **Set Presence Command**\n\n"+
					"*\"Let me show you how I'm feeling~\"* 💕\n\n"+
					"**Usage:**\n"+
					"`set-presence <type> <text>` - Set activity with text\n"+
					"`set-presence status <status>` - Set online status\n"+
					"`set-presence clear` - Clear current activity\n\n"+
					"**Activity Types:**\n"+
					"• `playing` - Playing <text>\n"+
					"• `watching` - Watching <text>\n"+
					"• `listening` - Listening to <text>\n"+
					"• `streaming` - Streaming <text> (requires URL)\n"+
					"• `competing` - Competing in <text>\n\n"+
					"**Status Options:**\n"+
					"• `online` - Green dot\n"+
					"• `idle` - Yellow dot\n"+
					"• `dnd` - Red dot (Do Not Disturb)\n"+
					"• `invisible` - Appear offline\n\n"+
					"**Examples:**\n"+
					"`set-presence playing with Yukki's heart`\n"+
					"`set-presence watching over my senpai`\n"+
					"`set-presence listening to Future Diary OST`\n"+
					"`set-presence streaming Yuno Gasai https://twitch.tv/example`\n"+
					"`set-presence status dnd`\n"+
					"`set-presence clear`")
		}
		return nil
	}

	subcommand := strings.ToLower(ctx.Args[0])

	// Handle clear
	if subcommand == "clear" || subcommand == "reset" || subcommand == "none" {
		err := ctx.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{},
			Status:     "online",
		})
		if err != nil {
			if ctx.Message != nil {
				ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
					fmt.Sprintf("❌ Failed to clear presence: %v", err))
			}
			return err
		}
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "✅ Presence cleared~")
		}
		return nil
	}

	// Handle status change
	if subcommand == "status" {
		if len(ctx.Args) < 2 {
			if ctx.Message != nil {
				ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
					"❌ Invalid status. Options: `online`, `idle`, `dnd`, `invisible`")
			}
			return nil
		}

		status := strings.ToLower(ctx.Args[1])
		validStatuses := map[string]bool{"online": true, "idle": true, "dnd": true, "invisible": true}
		if !validStatuses[status] {
			if ctx.Message != nil {
				ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
					"❌ Invalid status. Options: `online`, `idle`, `dnd`, `invisible`")
			}
			return nil
		}

		err := ctx.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: status,
		})
		if err != nil {
			if ctx.Message != nil {
				ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
					fmt.Sprintf("❌ Failed to set status: %v", err))
			}
			return err
		}

		statusEmoji := map[string]string{
			"online":    "🟢",
			"idle":      "🟡",
			"dnd":       "🔴",
			"invisible": "⚪",
		}

		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				fmt.Sprintf("%s Status set to **%s**~", statusEmoji[status], status))
		}
		return nil
	}

	// Handle activity type
	activityTypes := map[string]discordgo.ActivityType{
		"playing":   discordgo.ActivityTypeGame,
		"streaming": discordgo.ActivityTypeStreaming,
		"listening": discordgo.ActivityTypeListening,
		"watching":  discordgo.ActivityTypeWatching,
		"competing": discordgo.ActivityTypeCompeting,
	}

	actType, ok := activityTypes[subcommand]
	if !ok {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				fmt.Sprintf("❌ Unknown type: `%s`\nValid types: `playing`, `watching`, `listening`, `streaming`, `competing`", subcommand))
		}
		return nil
	}

	// Get the activity text
	if len(ctx.Args) < 2 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ Please provide activity text!")
		}
		return nil
	}

	activityText := strings.Join(ctx.Args[1:], " ")
	var streamURL string

	// For streaming, check for URL
	if actType == discordgo.ActivityTypeStreaming {
		// Extract URL from text
		parts := strings.Fields(activityText)
		for i, part := range parts {
			if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
				streamURL = part
				// Remove URL from activity text
				parts = append(parts[:i], parts[i+1:]...)
				activityText = strings.Join(parts, " ")
				break
			}
		}

		if streamURL == "" {
			if ctx.Message != nil {
				ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
					"❌ Streaming requires a URL!\nExample: `set-presence streaming My Stream https://twitch.tv/example`")
			}
			return nil
		}
	}

	if activityText == "" {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ Please provide activity text!")
		}
		return nil
	}

	activity := &discordgo.Activity{
		Name: activityText,
		Type: actType,
	}
	if streamURL != "" {
		activity.URL = streamURL
	}

	err := ctx.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{activity},
	})

	if err != nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				fmt.Sprintf("❌ Failed to set presence: %v", err))
		}
		return err
	}

	typeDisplay := strings.Title(subcommand)
	if ctx.Message != nil {
		msg := fmt.Sprintf("✅ Now **%s** %s", typeDisplay, activityText)
		if streamURL != "" {
			msg += fmt.Sprintf(" (%s)", streamURL)
		}
		msg += "~"
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, msg)
	} else {
		fmt.Printf("Now %s %s\n", subcommand, activityText)
	}

	return nil
}

// ConfigCommand shows server configuration
type ConfigCommand struct {
	DB interface {
		GetGuildPrefix(guildID string) (string, error)
		GetGuildConfig(guildID string) (map[string]interface{}, error)
	}
}

func (c *ConfigCommand) Name() string        { return "config" }
func (c *ConfigCommand) Aliases() []string   { return []string{"settings", "cfg"} }
func (c *ConfigCommand) Description() string { return "Show server configuration" }
func (c *ConfigCommand) Usage() string       { return "config" }
func (c *ConfigCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *ConfigCommand) MasterOnly() bool { return false }

func (c *ConfigCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	guild, err := ctx.Session.Guild(ctx.Message.GuildID)
	if err != nil {
		return err
	}

	prefix, _ := c.DB.GetGuildPrefix(ctx.Message.GuildID)
	if prefix == "" {
		prefix = ctx.GetPrefix()
	}

	config, _ := c.DB.GetGuildConfig(ctx.Message.GuildID)

	spamFilter := "Enabled"
	leveling := "Enabled"
	logChannel := "Not set"

	if config != nil {
		if sf, ok := config["spam_filter"].(bool); ok && !sf {
			spamFilter = "Disabled"
		}
		if lv, ok := config["leveling"].(bool); ok && !lv {
			leveling = "Disabled"
		}
		if lc, ok := config["log_channel"].(string); ok && lc != "" {
			logChannel = fmt.Sprintf("<#%s>", lc)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Configuration for %s", guild.Name),
		Color: 0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Prefix", Value: fmt.Sprintf("`%s`", prefix), Inline: true},
			{Name: "Spam Filter", Value: spamFilter, Inline: true},
			{Name: "Leveling", Value: leveling, Inline: true},
			{Name: "Log Channel", Value: logChannel, Inline: true},
		},
	}

	if guild.Icon != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.png", guild.ID, guild.Icon),
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// InitGuildCommand initializes guild settings
type InitGuildCommand struct {
	DB interface {
		InitGuild(guildID string) error
	}
}

func (c *InitGuildCommand) Name() string        { return "init-guild" }
func (c *InitGuildCommand) Aliases() []string   { return []string{"initguild", "setup"} }
func (c *InitGuildCommand) Description() string { return "Initialize guild settings and database tables" }
func (c *InitGuildCommand) Usage() string       { return "init-guild" }
func (c *InitGuildCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *InitGuildCommand) MasterOnly() bool { return false }

func (c *InitGuildCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	guild, err := ctx.Session.Guild(ctx.Message.GuildID)
	if err != nil {
		return err
	}

	if err := c.DB.InitGuild(ctx.Message.GuildID); err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error initializing guild settings.")
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Guild Initialized",
		Description: fmt.Sprintf("**%s** has been set up!", guild.Name),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Prefix", Value: fmt.Sprintf("`%s`", ctx.GetPrefix()), Inline: true},
			{Name: "Leveling", Value: "Enabled", Inline: true},
			{Name: "Spam Filter", Value: "Enabled", Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Use %sconfig to view all settings", ctx.GetPrefix()),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetSpamFilterCommand toggles spam filter
type SetSpamFilterCommand struct {
	DB interface {
		SetGuildSpamFilter(guildID string, enabled bool) error
	}
}

func (c *SetSpamFilterCommand) Name() string        { return "set-spamfilter" }
func (c *SetSpamFilterCommand) Aliases() []string   { return []string{"spamfilter", "togglespam"} }
func (c *SetSpamFilterCommand) Description() string { return "Enable or disable the spam filter" }
func (c *SetSpamFilterCommand) Usage() string       { return "set-spamfilter <on|off>" }
func (c *SetSpamFilterCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetSpamFilterCommand) MasterOnly() bool { return false }

func (c *SetSpamFilterCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Usage: `set-spamfilter on` or `set-spamfilter off`")
		return nil
	}

	state := strings.ToLower(ctx.Args[0])
	var enabled bool
	switch state {
	case "on", "enable", "true", "1":
		enabled = true
	case "off", "disable", "false", "0":
		enabled = false
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Usage: `set-spamfilter on` or `set-spamfilter off`")
		return nil
	}

	if err := c.DB.SetGuildSpamFilter(ctx.Message.GuildID, enabled); err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error updating spam filter setting.")
		return err
	}

	status := "enabled"
	color := 0x00FF00
	if !enabled {
		status = "disabled"
		color = 0xFFAA00
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Spam Filter Updated",
		Description: fmt.Sprintf("Spam filter has been **%s**", status),
		Color:       color,
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetLevelingCommand toggles leveling system
type SetLevelingCommand struct {
	DB interface {
		SetGuildLeveling(guildID string, enabled bool) error
	}
}

func (c *SetLevelingCommand) Name() string        { return "set-leveling" }
func (c *SetLevelingCommand) Aliases() []string   { return []string{"toggleleveling", "setxp"} }
func (c *SetLevelingCommand) Description() string { return "Enable or disable the leveling system" }
func (c *SetLevelingCommand) Usage() string       { return "set-leveling <on|off>" }
func (c *SetLevelingCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetLevelingCommand) MasterOnly() bool { return false }

func (c *SetLevelingCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Usage: `set-leveling on` or `set-leveling off`")
		return nil
	}

	state := strings.ToLower(ctx.Args[0])
	var enabled bool
	switch state {
	case "on", "enable", "true", "1":
		enabled = true
	case "off", "disable", "false", "0":
		enabled = false
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Usage: `set-leveling on` or `set-leveling off`")
		return nil
	}

	if err := c.DB.SetGuildLeveling(ctx.Message.GuildID, enabled); err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error updating leveling setting.")
		return err
	}

	status := "enabled"
	color := 0x00FF00
	if !enabled {
		status = "disabled"
		color = 0xFFAA00
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Leveling System Updated",
		Description: fmt.Sprintf("Leveling has been **%s**", status),
		Color:       color,
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
