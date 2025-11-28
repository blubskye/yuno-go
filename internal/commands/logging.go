package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// LoggingConfigInterface defines what logging commands need from the bot
type LoggingConfigInterface interface {
	GetDB() DBInterface
	GetLoggingConfig(guildID string) (interface{}, error)
}

// SetLogChannelCommand sets the log channel for a guild
type SetLogChannelCommand struct{}

func (c *SetLogChannelCommand) Name() string        { return "setlogchannel" }
func (c *SetLogChannelCommand) Aliases() []string   { return []string{"logchannel"} }
func (c *SetLogChannelCommand) Description() string { return "Set the channel for logging events" }
func (c *SetLogChannelCommand) Usage() string       { return "setlogchannel <#channel>" }
func (c *SetLogChannelCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetLogChannelCommand) MasterOnly() bool { return false }

func (c *SetLogChannelCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	// Parse channel mention
	channelID := strings.Trim(ctx.Args[0], "<>#")

	// Verify channel exists
	channel, err := ctx.Session.Channel(channelID)
	if err != nil || channel.GuildID != ctx.Message.GuildID {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid channel! Please mention a valid channel in this server.")
		return nil
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Update or insert logging config
	_, err = db.Exec(`
		INSERT INTO logging_config (guild_id, log_channel_id, enabled)
		VALUES (?, ?, 1)
		ON CONFLICT(guild_id) DO UPDATE SET log_channel_id = ?, enabled = 1`,
		ctx.Message.GuildID, channelID, channelID,
	)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to set log channel: "+err.Error())
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("Logging channel set to <#%s>! Logging is now enabled.", channelID))
	return nil
}

// ToggleLoggingCommand enables or disables logging for a guild
type ToggleLoggingCommand struct{}

func (c *ToggleLoggingCommand) Name() string        { return "togglelogging" }
func (c *ToggleLoggingCommand) Aliases() []string   { return []string{"logging"} }
func (c *ToggleLoggingCommand) Description() string { return "Enable or disable logging" }
func (c *ToggleLoggingCommand) Usage() string       { return "togglelogging <on|off>" }
func (c *ToggleLoggingCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *ToggleLoggingCommand) MasterOnly() bool { return false }

func (c *ToggleLoggingCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	enabled := 0
	switch strings.ToLower(ctx.Args[0]) {
	case "on", "enable", "enabled", "true", "1":
		enabled = 1
	case "off", "disable", "disabled", "false", "0":
		enabled = 0
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	_, err := db.Exec(`
		INSERT INTO logging_config (guild_id, enabled)
		VALUES (?, ?)
		ON CONFLICT(guild_id) DO UPDATE SET enabled = ?`,
		ctx.Message.GuildID, enabled, enabled,
	)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to toggle logging: "+err.Error())
		return err
	}

	status := "disabled"
	if enabled == 1 {
		status = "enabled"
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("Logging has been %s!", status))
	return nil
}

// ConfigureLogTypeCommand configures specific log types
type ConfigureLogTypeCommand struct{}

func (c *ConfigureLogTypeCommand) Name() string      { return "configlog" }
func (c *ConfigureLogTypeCommand) Aliases() []string { return []string{"logconfig"} }
func (c *ConfigureLogTypeCommand) Description() string {
	return "Configure specific logging types"
}
func (c *ConfigureLogTypeCommand) Usage() string {
	return "configlog <type> <on|off>\nTypes: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, presence"
}
func (c *ConfigureLogTypeCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *ConfigureLogTypeCommand) MasterOnly() bool { return false }

func (c *ConfigureLogTypeCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	logType := strings.ToLower(ctx.Args[0])
	enabled := 0
	switch strings.ToLower(ctx.Args[1]) {
	case "on", "enable", "enabled", "true", "1":
		enabled = 1
	case "off", "disable", "disabled", "false", "0":
		enabled = 0
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Second argument must be on or off")
		return nil
	}

	// Map friendly names to database columns
	columnName := ""
	friendlyName := ""
	switch logType {
	case "message_delete", "msgdelete", "delete":
		columnName = "message_delete"
		friendlyName = "Message Deletion"
	case "message_edit", "msgedit", "edit":
		columnName = "message_edit"
		friendlyName = "Message Editing"
	case "voice_join", "join", "voicejoin":
		columnName = "member_join_voice"
		friendlyName = "Voice Channel Join"
	case "voice_leave", "leave", "voiceleave":
		columnName = "member_leave_voice"
		friendlyName = "Voice Channel Leave"
	case "nickname", "nick":
		columnName = "nickname_change"
		friendlyName = "Nickname Changes"
	case "avatar", "pfp":
		columnName = "avatar_change"
		friendlyName = "Avatar Changes"
	case "presence", "status":
		columnName = "presence_change"
		friendlyName = "Presence Changes"
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid log type! Use: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, or presence")
		return nil
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	query := fmt.Sprintf(`
		INSERT INTO logging_config (guild_id, %s)
		VALUES (?, ?)
		ON CONFLICT(guild_id) DO UPDATE SET %s = ?`,
		columnName, columnName,
	)

	_, err := db.Exec(query, ctx.Message.GuildID, enabled, enabled)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to configure log type: "+err.Error())
		return err
	}

	status := "disabled"
	if enabled == 1 {
		status = "enabled"
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("%s logging has been %s!", friendlyName, status))
	return nil
}

// SetPresenceBatchCommand sets the presence batch interval
type SetPresenceBatchCommand struct{}

func (c *SetPresenceBatchCommand) Name() string        { return "setpresencebatch" }
func (c *SetPresenceBatchCommand) Aliases() []string   { return []string{"presencebatch"} }
func (c *SetPresenceBatchCommand) Description() string { return "Set presence batch interval" }
func (c *SetPresenceBatchCommand) Usage() string {
	return "setpresencebatch <seconds> (min: 30, max: 300)"
}
func (c *SetPresenceBatchCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SetPresenceBatchCommand) MasterOnly() bool { return false }

func (c *SetPresenceBatchCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	seconds, err := strconv.Atoi(ctx.Args[0])
	if err != nil || seconds < 30 || seconds > 300 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Please provide a valid number between 30 and 300 seconds.")
		return nil
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	_, err = db.Exec(`
		INSERT INTO logging_config (guild_id, presence_batch_seconds)
		VALUES (?, ?)
		ON CONFLICT(guild_id) DO UPDATE SET presence_batch_seconds = ?`,
		ctx.Message.GuildID, seconds, seconds,
	)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to set presence batch interval: "+err.Error())
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("Presence batch interval set to %d seconds!", seconds))
	return nil
}

// DisableChannelLoggingCommand disables specific logging types for a channel
type DisableChannelLoggingCommand struct{}

func (c *DisableChannelLoggingCommand) Name() string      { return "disablechannellog" }
func (c *DisableChannelLoggingCommand) Aliases() []string { return []string{"nolog"} }
func (c *DisableChannelLoggingCommand) Description() string {
	return "Disable specific logging types for a channel"
}
func (c *DisableChannelLoggingCommand) Usage() string {
	return "disablechannellog <#channel> <type>\nTypes: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, presence"
}
func (c *DisableChannelLoggingCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *DisableChannelLoggingCommand) MasterOnly() bool { return false }

func (c *DisableChannelLoggingCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	// Parse channel mention
	channelID := strings.Trim(ctx.Args[0], "<>#")

	// Verify channel exists
	channel, err := ctx.Session.Channel(channelID)
	if err != nil || channel.GuildID != ctx.Message.GuildID {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid channel! Please mention a valid channel in this server.")
		return nil
	}

	logType := strings.ToLower(ctx.Args[1])

	// Map friendly names to database columns
	columnName := ""
	friendlyName := ""
	switch logType {
	case "message_delete", "msgdelete", "delete":
		columnName = "message_delete"
		friendlyName = "Message Deletion"
	case "message_edit", "msgedit", "edit":
		columnName = "message_edit"
		friendlyName = "Message Editing"
	case "voice_join", "join", "voicejoin":
		columnName = "member_join_voice"
		friendlyName = "Voice Channel Join"
	case "voice_leave", "leave", "voiceleave":
		columnName = "member_leave_voice"
		friendlyName = "Voice Channel Leave"
	case "nickname", "nick":
		columnName = "nickname_change"
		friendlyName = "Nickname Changes"
	case "avatar", "pfp":
		columnName = "avatar_change"
		friendlyName = "Avatar Changes"
	case "presence", "status":
		columnName = "presence_change"
		friendlyName = "Presence Changes"
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid log type! Use: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, or presence")
		return nil
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	query := fmt.Sprintf(`
		INSERT INTO logging_channel_overrides (guild_id, channel_id, %s)
		VALUES (?, ?, 0)
		ON CONFLICT(guild_id, channel_id) DO UPDATE SET %s = 0`,
		columnName, columnName,
	)

	_, err = db.Exec(query, ctx.Message.GuildID, channelID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to disable channel logging: "+err.Error())
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("%s logging disabled for <#%s>!", friendlyName, channelID))
	return nil
}

// EnableChannelLoggingCommand enables specific logging types for a channel
type EnableChannelLoggingCommand struct{}

func (c *EnableChannelLoggingCommand) Name() string      { return "enablechannellog" }
func (c *EnableChannelLoggingCommand) Aliases() []string { return []string{"yeslog"} }
func (c *EnableChannelLoggingCommand) Description() string {
	return "Enable specific logging types for a channel"
}
func (c *EnableChannelLoggingCommand) Usage() string {
	return "enablechannellog <#channel> <type>\nTypes: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, presence"
}
func (c *EnableChannelLoggingCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *EnableChannelLoggingCommand) MasterOnly() bool { return false }

func (c *EnableChannelLoggingCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: "+c.Usage())
		return nil
	}

	// Parse channel mention
	channelID := strings.Trim(ctx.Args[0], "<>#")

	// Verify channel exists
	channel, err := ctx.Session.Channel(channelID)
	if err != nil || channel.GuildID != ctx.Message.GuildID {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid channel! Please mention a valid channel in this server.")
		return nil
	}

	logType := strings.ToLower(ctx.Args[1])

	// Map friendly names to database columns
	columnName := ""
	friendlyName := ""
	switch logType {
	case "message_delete", "msgdelete", "delete":
		columnName = "message_delete"
		friendlyName = "Message Deletion"
	case "message_edit", "msgedit", "edit":
		columnName = "message_edit"
		friendlyName = "Message Editing"
	case "voice_join", "join", "voicejoin":
		columnName = "member_join_voice"
		friendlyName = "Voice Channel Join"
	case "voice_leave", "leave", "voiceleave":
		columnName = "member_leave_voice"
		friendlyName = "Voice Channel Leave"
	case "nickname", "nick":
		columnName = "nickname_change"
		friendlyName = "Nickname Changes"
	case "avatar", "pfp":
		columnName = "avatar_change"
		friendlyName = "Avatar Changes"
	case "presence", "status":
		columnName = "presence_change"
		friendlyName = "Presence Changes"
	default:
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Invalid log type! Use: message_delete, message_edit, voice_join, voice_leave, nickname, avatar, or presence")
		return nil
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	query := fmt.Sprintf(`
		INSERT INTO logging_channel_overrides (guild_id, channel_id, %s)
		VALUES (?, ?, 1)
		ON CONFLICT(guild_id, channel_id) DO UPDATE SET %s = 1`,
		columnName, columnName,
	)

	_, err = db.Exec(query, ctx.Message.GuildID, channelID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to enable channel logging: "+err.Error())
		return err
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("%s logging enabled for <#%s>!", friendlyName, channelID))
	return nil
}

// LogStatusCommand shows the current logging configuration
type LogStatusCommand struct{}

func (c *LogStatusCommand) Name() string        { return "logstatus" }
func (c *LogStatusCommand) Aliases() []string   { return []string{"loggingstatus"} }
func (c *LogStatusCommand) Description() string { return "Show current logging configuration" }
func (c *LogStatusCommand) Usage() string       { return "logstatus" }
func (c *LogStatusCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *LogStatusCommand) MasterOnly() bool { return false }

func (c *LogStatusCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	var (
		logChannelID                                                            string
		msgDel, msgEdit, joinVoice, leaveVoice, nick, avatar, presence, enabled int
		presenceBatchSeconds                                                    int
	)

	err := db.QueryRow(`
		SELECT log_channel_id, message_delete, message_edit, member_join_voice,
		       member_leave_voice, nickname_change, avatar_change, presence_change,
		       presence_batch_seconds, enabled
		FROM logging_config WHERE guild_id = ?`, ctx.Message.GuildID).Scan(
		&logChannelID, &msgDel, &msgEdit, &joinVoice, &leaveVoice,
		&nick, &avatar, &presence, &presenceBatchSeconds, &enabled,
	)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Logging is not configured for this server. Use `setlogchannel` to configure it.")
		return nil
	}

	status := "disabled"
	if enabled == 1 {
		status = "enabled"
	}

	logChannelText := logChannelID
	if logChannelID != "" {
		logChannelText = fmt.Sprintf("<#%s>", logChannelID)
	}

	embed := &discordgo.MessageEmbed{
		Title: "Logging Configuration",
		Color: 0x3498db,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  status,
				Inline: true,
			},
			{
				Name:   "Log Channel",
				Value:  logChannelText,
				Inline: true,
			},
			{
				Name:   "Presence Batch Interval",
				Value:  fmt.Sprintf("%d seconds", presenceBatchSeconds),
				Inline: true,
			},
		},
	}

	// Add logging types
	types := []struct {
		name    string
		enabled int
	}{
		{"Message Delete", msgDel},
		{"Message Edit", msgEdit},
		{"Voice Join", joinVoice},
		{"Voice Leave", leaveVoice},
		{"Nickname Change", nick},
		{"Avatar Change", avatar},
		{"Presence Change", presence},
	}

	var enabledTypes []string
	var disabledTypes []string

	for _, t := range types {
		if t.enabled == 1 {
			enabledTypes = append(enabledTypes, t.name)
		} else {
			disabledTypes = append(disabledTypes, t.name)
		}
	}

	if len(enabledTypes) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Enabled Types",
			Value:  strings.Join(enabledTypes, ", "),
			Inline: false,
		})
	}

	if len(disabledTypes) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Disabled Types",
			Value:  strings.Join(disabledTypes, ", "),
			Inline: false,
		})
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
