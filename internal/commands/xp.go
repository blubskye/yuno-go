package commands

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// DBInterface defines what commands need from the database
type DBInterface interface {
	QueryRow(query string, args ...interface{}) interface{ Scan(dest ...interface{}) error }
	Exec(query string, args ...interface{}) (interface{}, error)
}

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

	// Get database from bot
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Query XP data directly from database
	var xp, level int
	err := db.QueryRow(`
		SELECT exp, level FROM glevel 
		WHERE guild_id = ? AND user_id = ?`,
		ctx.Message.GuildID, targetUser.ID).Scan(&xp, &level)
	
	if err != nil {
		// User not found - they have 0 XP
		xp = 0
		level = 0
	}

	// Calculate XP needed for next level
	neededXP := 5*level*level + 50*level + 100
	remaining := neededXP - xp

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
				Value:  fmt.Sprintf("%d", level),
				Inline: true,
			},
			{
				Name:   "Current exp",
				Value:  fmt.Sprintf("%d", xp),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Exp needed until next level (%d)", level+1),
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

	// Get database from bot
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Set level with 0 XP
	_, err = db.Exec(`INSERT OR REPLACE INTO glevel (guild_id, user_id, exp, level, enabled) 
		VALUES (?, ?, 0, ?, 'enabled')`,
		ctx.Message.GuildID, targetUser.ID, level)
	
	if err != nil {
		return err
	}

	neededXP := 5*level*level + 50*level + 100

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
				Value:  fmt.Sprintf("%d", level),
				Inline: true,
			},
			{
				Name:   "Current exp",
				Value:  "0",
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Exp needed until next level (%d)", level+1),
				Value:  fmt.Sprintf("%d", neededXP),
				Inline: false,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
