package commands

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"yuno-bot/internal/database"
)

// XPCommand shows XP information
type XPCommand struct{}

func (c *XPCommand) Name() string        { return "xp" }
func (c *XPCommand) Aliases() []string   { return []string{"rank", "level", "exp"} }
func (c *XPCommand) Description() string { return "Show your XP and level" }
func (c *XPCommand) Usage() string       { return "xp [@user]" }
func (c *XPCommand) RequiredPermissions() []int64 { return nil }
func (c *XPCommand) MasterOnly() bool    { return false }

func (c *XPCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	// Get target user (mention or author)
	targetUser := ctx.Message.Author
	if len(ctx.Message.Mentions) > 0 {
		targetUser = ctx.Message.Mentions[0]
	}

	if targetUser.Bot {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "🤖 Bots don't have XP!")
		return nil
	}

	// Get bot instance
	bot := ctx.Bot.(interface {
		GetDB() *database.Database
	})
	db := bot.GetDB()

	// Get XP data
	xpData, err := db.GetXPData(ctx.Message.GuildID, targetUser.ID)
	if err != nil {
		return err
	}

	// Calculate XP needed for next level
	neededXP := 5*xpData.Level*xpData.Level + 50*xpData.Level + 100
	remaining := neededXP - xpData.XP

	// Get avatar URL
	avatarURL := targetUser.AvatarURL("256")

	// Create embed
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.Username + "'s experience card",
			IconURL: avatarURL,
		},
		Color: 0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current level",
				Value:  fmt.Sprintf("%d", xpData.Level),
				Inline: true,
			},
			{
				Name:   "Current exp",
				Value:  fmt.Sprintf("%d", xpData.XP),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Exp needed until next level (%d)", xpData.Level+1),
				Value:  fmt.Sprintf("%d", remaining),
				Inline: false,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SetLevelCommand sets a user's level (master only)
type SetLevelCommand struct{}

func (c *SetLevelCommand) Name() string        { return "set-level" }
func (c *SetLevelCommand) Aliases() []string   { return []string{"slvl", "setlevel"} }
func (c *SetLevelCommand) Description() string { return "Set a user's level" }
func (c *SetLevelCommand) Usage() string       { return "set-level <level> [@user]" }
func (c *SetLevelCommand) RequiredPermissions() []int64 { return nil }
func (c *SetLevelCommand) MasterOnly() bool    { return true }

func (c *SetLevelCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Usage: `set-level <level> [@user]`")
		return nil
	}

	// Parse level
	level, err := strconv.Atoi(ctx.Args[0])
	if err != nil || level < 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Level must be a positive number")
		return nil
	}

	// Get target user
	targetUser := ctx.Message.Author
	if len(ctx.Message.Mentions) > 0 {
		targetUser = ctx.Message.Mentions[0]
	}

	if targetUser.Bot {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "🤖 Bots don't have XP!")
		return nil
	}

	// Get bot instance
	bot := ctx.Bot.(interface {
		GetDB() *database.Database
	})
	db := bot.GetDB()

	// Set level with 0 XP
	err = db.SetXPData(ctx.Message.GuildID, targetUser.ID, 0, level)
	if err != nil {
		return err
	}

	// Get updated data
	xpData, err := db.GetXPData(ctx.Message.GuildID, targetUser.ID)
	if err != nil {
		return err
	}

	neededXP := 5*xpData.Level*xpData.Level + 50*xpData.Level + 100

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    targetUser.Username + "'s experience card",
			IconURL: targetUser.AvatarURL("256"),
		},
		Title: "Level has been changed.",
		Color: 0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current level",
				Value:  fmt.Sprintf("%d", xpData.Level),
				Inline: true,
			},
			{
				Name:   "Current exp",
				Value:  fmt.Sprintf("%d", xpData.XP),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Exp needed until next level (%d)", xpData.Level+1),
				Value:  fmt.Sprintf("%d", neededXP-xpData.XP),
				Inline: false,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}