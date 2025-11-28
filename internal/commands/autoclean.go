package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AutoCleanCommand manages auto-cleaning channels
type AutoCleanCommand struct{}

func (c *AutoCleanCommand) Name() string        { return "auto-clean" }
func (c *AutoCleanCommand) Aliases() []string   { return []string{"ac", "autoclean"} }
func (c *AutoCleanCommand) Description() string { return "Manage automatic channel cleaning" }
func (c *AutoCleanCommand) Usage() string {
	return "auto-clean <add|remove|list> [#channel] [hours] [warning_minutes]"
}
func (c *AutoCleanCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageChannels}
}
func (c *AutoCleanCommand) MasterOnly() bool { return false }

func (c *AutoCleanCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		return c.showHelp(ctx)
	}

	subcommand := strings.ToLower(ctx.Args[0])

	switch subcommand {
	case "add":
		return c.addAutoClean(ctx)
	case "remove", "delete", "del":
		return c.removeAutoClean(ctx)
	case "list":
		return c.listAutoClean(ctx)
	default:
		return c.showHelp(ctx)
	}
}

func (c *AutoCleanCommand) showHelp(ctx *Context) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Auto-Clean Commands",
		Description: "Automatically clean channels by cloning them at scheduled intervals",
		Color:       0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Add Auto-Clean",
				Value: "`auto-clean add #channel <hours> [warning_minutes]`\nExample: `auto-clean add #main 24 15`",
			},
			{
				Name:  "Remove Auto-Clean",
				Value: "`auto-clean remove #channel`",
			},
			{
				Name:  "List Auto-Cleans",
				Value: "`auto-clean list`",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "The bot will warn X minutes before cleaning and preserve channel settings",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AutoCleanCommand) addAutoClean(ctx *Context) error {
	// Usage: auto-clean add #channel hours [warning_minutes]
	if len(ctx.Args) < 3 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `auto-clean add #channel <hours> [warning_minutes]`")
		return nil
	}

	// Get channel
	if len(ctx.Message.MentionChannels) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Please mention a channel to auto-clean")
		return nil
	}
	channel := ctx.Message.MentionChannels[0]

	// Parse hours
	hours, err := strconv.Atoi(ctx.Args[2])
	if err != nil || hours <= 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Hours must be a positive number")
		return nil
	}

	// Parse warning minutes (default: 15)
	warningMinutes := 15
	if len(ctx.Args) >= 4 {
		warningMinutes, err = strconv.Atoi(ctx.Args[3])
		if err != nil || warningMinutes < 0 {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ Warning minutes must be a positive number")
			return nil
		}
	}

	// Calculate next run time
	nextRun := time.Now().Add(time.Duration(hours) * time.Hour)

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Insert or replace auto-clean config
	_, err = db.Exec(`
		INSERT OR REPLACE INTO autoclean 
		(guild_id, channel_id, interval_hours, warning_minutes, next_run, enabled, custom_message, custom_image) 
		VALUES (?, ?, ?, ?, ?, 1, '', '')`,
		ctx.Message.GuildID, channel.ID, hours, warningMinutes, nextRun.Format(time.RFC3339))

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to add auto-clean: %v", err))
		return err
	}

	// Success embed
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Auto-Clean Added",
		Description: fmt.Sprintf("Channel <#%s> will be automatically cleaned", channel.ID),
		Color:       0x43CC24,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Interval",
				Value:  fmt.Sprintf("%d hours", hours),
				Inline: true,
			},
			{
				Name:   "Warning Time",
				Value:  fmt.Sprintf("%d minutes before", warningMinutes),
				Inline: true,
			},
			{
				Name:   "Next Clean",
				Value:  fmt.Sprintf("<t:%d:R>", nextRun.Unix()),
				Inline: false,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AutoCleanCommand) removeAutoClean(ctx *Context) error {
	if len(ctx.Message.MentionChannels) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Please mention a channel to remove from auto-clean")
		return nil
	}
	channel := ctx.Message.MentionChannels[0]

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Delete auto-clean config
	_, err := db.Exec(`
		DELETE FROM autoclean 
		WHERE guild_id = ? AND channel_id = ?`,
		ctx.Message.GuildID, channel.ID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to remove auto-clean: %v", err))
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Auto-Clean Removed",
		Description: fmt.Sprintf("Channel <#%s> will no longer be automatically cleaned", channel.ID),
		Color:       0x43CC24,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AutoCleanCommand) listAutoClean(ctx *Context) error {
	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Query all auto-clean configs for this guild
	type AutoCleanRow interface {
		Scan(dest ...interface{}) error
	}
	
	rows, err := db.(interface {
		Query(query string, args ...interface{}) (interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
		}, error)
	}).Query(`
		SELECT channel_id, interval_hours, warning_minutes, next_run, enabled 
		FROM autoclean 
		WHERE guild_id = ?
		ORDER BY next_run ASC`,
		ctx.Message.GuildID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to list auto-cleans: %v", err))
		return err
	}
	defer rows.Close()

	embed := &discordgo.MessageEmbed{
		Title: "Auto-Clean Schedule",
		Color: 0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{},
	}

	count := 0
	for rows.Next() {
		var channelID string
		var intervalHours, warningMinutes int
		var nextRunStr string
		var enabled bool

		if err := rows.Scan(&channelID, &intervalHours, &warningMinutes, &nextRunStr, &enabled); err != nil {
			continue
		}

		nextRun, _ := time.Parse(time.RFC3339, nextRunStr)
		status := "✅ Active"
		if !enabled {
			status = "⏸️ Paused"
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("%s <#%s>", status, channelID),
			Value: fmt.Sprintf(
				"**Interval:** %d hours\n**Warning:** %d min\n**Next:** <t:%d:R>",
				intervalHours, warningMinutes, nextRun.Unix(),
			),
			Inline: true,
		})
		count++
	}

	if count == 0 {
		embed.Description = "No auto-clean schedules configured for this server."
	} else {
		embed.Description = fmt.Sprintf("Found %d scheduled channel(s)", count)
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetCleanMessageCommand sets custom message for auto-clean
type SetCleanMessageCommand struct{}

func (c *SetCleanMessageCommand) Name() string        { return "set-clean-message" }
func (c *SetCleanMessageCommand) Aliases() []string   { return []string{"scm"} }
func (c *SetCleanMessageCommand) Description() string { return "Set custom message after cleaning" }
func (c *SetCleanMessageCommand) Usage() string {
	return "set-clean-message #channel <message>"
}
func (c *SetCleanMessageCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageChannels}
}
func (c *SetCleanMessageCommand) MasterOnly() bool { return false }

func (c *SetCleanMessageCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Message.MentionChannels) == 0 || len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `set-clean-message #channel <message>`")
		return nil
	}

	channel := ctx.Message.MentionChannels[0]
	message := strings.Join(ctx.Args[1:], " ")

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Update custom message
	_, err := db.Exec(`
		UPDATE autoclean 
		SET custom_message = ? 
		WHERE guild_id = ? AND channel_id = ?`,
		message, ctx.Message.GuildID, channel.ID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to set message: %v", err))
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Custom Message Set",
		Description: fmt.Sprintf("Custom message for <#%s> has been updated", channel.ID),
		Color:       0x43CC24,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Message Preview",
				Value: message,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetCleanImageCommand sets custom image for auto-clean
type SetCleanImageCommand struct{}

func (c *SetCleanImageCommand) Name() string        { return "set-clean-image" }
func (c *SetCleanImageCommand) Aliases() []string   { return []string{"sci"} }
func (c *SetCleanImageCommand) Description() string { return "Set custom image after cleaning" }
func (c *SetCleanImageCommand) Usage() string {
	return "set-clean-image #channel <image_url>"
}
func (c *SetCleanImageCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageChannels}
}
func (c *SetCleanImageCommand) MasterOnly() bool { return false }

func (c *SetCleanImageCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Message.MentionChannels) == 0 || len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `set-clean-image #channel <image_url>`")
		return nil
	}

	channel := ctx.Message.MentionChannels[0]
	imageURL := ctx.Args[1]

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Update custom image
	_, err := db.Exec(`
		UPDATE autoclean 
		SET custom_image = ? 
		WHERE guild_id = ? AND channel_id = ?`,
		imageURL, ctx.Message.GuildID, channel.ID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to set image: %v", err))
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Custom Image Set",
		Description: fmt.Sprintf("Custom image for <#%s> has been updated", channel.ID),
		Color:       0x43CC24,
		Image: &discordgo.MessageEmbedImage{
			URL: imageURL,
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
