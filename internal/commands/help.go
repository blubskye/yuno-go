package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// HelpCommand shows all available commands
type HelpCommand struct{}

func (c *HelpCommand) Name() string        { return "help" }
func (c *HelpCommand) Aliases() []string   { return []string{"h", "commands", "?"} }
func (c *HelpCommand) Description() string { return "Show all available commands" }
func (c *HelpCommand) Usage() string       { return "help [command]" }
func (c *HelpCommand) RequiredPermissions() []int64 { return nil }
func (c *HelpCommand) MasterOnly() bool    { return false }

func (c *HelpCommand) Execute(ctx *Context) error {
	// Get bot prefix from config
	prefix := ctx.GetPrefix()
	if prefix == "" {
		prefix = "?"
	}

	// If specific command requested
	if len(ctx.Args) > 0 {
		return c.showCommandHelp(ctx, ctx.Args[0], prefix)
	}

	// Get all registered commands
	allCommands := ctx.GetAllCommands()

	// Categorize commands
	basicCmds := []Command{}
	levelingCmds := []Command{}
	moderationCmds := []Command{}
	ownerCmds := []Command{}

	for _, cmd := range allCommands {
		if cmd.MasterOnly() {
			ownerCmds = append(ownerCmds, cmd)
		} else if strings.Contains(cmd.Name(), "level") || strings.Contains(cmd.Name(), "xp") || cmd.Name() == "rank" {
			levelingCmds = append(levelingCmds, cmd)
		} else if cmd.Name() == "ban" || cmd.Name() == "kick" {
			moderationCmds = append(moderationCmds, cmd)
		} else {
			basicCmds = append(basicCmds, cmd)
		}
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "Yuno Gasai - Command List",
		Description: fmt.Sprintf("Use `%shelp <command>` for detailed information about a command.\n\n", prefix),
		Color:       0xFF51FF,
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Prefix: %s | Total Commands: %d", prefix, len(allCommands)),
		},
	}

	// Add basic commands
	if len(basicCmds) > 0 {
		var cmdList []string
		for _, cmd := range basicCmds {
			cmdList = append(cmdList, fmt.Sprintf("`%s`", cmd.Name()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📌 Basic Commands",
			Value:  strings.Join(cmdList, ", "),
			Inline: false,
		})
	}

	// Add leveling commands
	if len(levelingCmds) > 0 {
		var cmdList []string
		for _, cmd := range levelingCmds {
			cmdList = append(cmdList, fmt.Sprintf("`%s`", cmd.Name()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⭐ Leveling Commands",
			Value:  strings.Join(cmdList, ", "),
			Inline: false,
		})
	}

	// Add moderation commands
	if len(moderationCmds) > 0 {
		var cmdList []string
		for _, cmd := range moderationCmds {
			cmdList = append(cmdList, fmt.Sprintf("`%s`", cmd.Name()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔨 Moderation Commands",
			Value:  strings.Join(cmdList, ", "),
			Inline: false,
		})
	}

	// Add owner commands (if user is owner)
	if len(ownerCmds) > 0 && c.isOwner(ctx) {
		var cmdList []string
		for _, cmd := range ownerCmds {
			cmdList = append(cmdList, fmt.Sprintf("`%s`", cmd.Name()))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👑 Owner Commands",
			Value:  strings.Join(cmdList, ", "),
			Inline: false,
		})
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	return nil
}

func (c *HelpCommand) showCommandHelp(ctx *Context, cmdName string, prefix string) error {
	// Find the command
	allCommands := ctx.GetAllCommands()
	
	var targetCmd Command
	for _, cmd := range allCommands {
		if cmd.Name() == cmdName {
			targetCmd = cmd
			break
		}
		// Check aliases
		for _, alias := range cmd.Aliases() {
			if alias == cmdName {
				targetCmd = cmd
				break
			}
		}
		if targetCmd != nil {
			break
		}
	}

	if targetCmd == nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, 
				fmt.Sprintf("❌ Command `%s` not found. Use `%shelp` to see all commands.", cmdName, prefix))
		}
		return nil
	}

	// Build detailed help embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Command: %s", targetCmd.Name()),
		Description: targetCmd.Description(),
		Color:       0xFF51FF,
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Usage
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Usage",
		Value:  fmt.Sprintf("`%s%s`", prefix, targetCmd.Usage()),
		Inline: false,
	})

	// Aliases
	if len(targetCmd.Aliases()) > 0 {
		aliases := []string{}
		for _, alias := range targetCmd.Aliases() {
			aliases = append(aliases, fmt.Sprintf("`%s`", alias))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Aliases",
			Value:  strings.Join(aliases, ", "),
			Inline: false,
		})
	}

	// Permissions required
	if perms := targetCmd.RequiredPermissions(); len(perms) > 0 {
		permNames := []string{}
		for _, perm := range perms {
			permNames = append(permNames, getPermissionName(perm))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Required Permissions",
			Value:  strings.Join(permNames, ", "),
			Inline: false,
		})
	}

	// Owner only
	if targetCmd.MasterOnly() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Restriction",
			Value:  "This command can only be used by bot owners.",
			Inline: false,
		})
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	return nil
}

func (c *HelpCommand) isOwner(ctx *Context) bool {
	if ctx.Message == nil {
		return false
	}
	
	return ctx.IsOwner()
}

func getPermissionName(perm int64) string {
	switch perm {
	case discordgo.PermissionKickMembers:
		return "Kick Members"
	case discordgo.PermissionBanMembers:
		return "Ban Members"
	case discordgo.PermissionAdministrator:
		return "Administrator"
	case discordgo.PermissionManageChannels:
		return "Manage Channels"
	case discordgo.PermissionManageServer:
		return "Manage Server"
	case discordgo.PermissionManageMessages:
		return "Manage Messages"
	case discordgo.PermissionManageRoles:
		return "Manage Roles"
	case discordgo.PermissionManageWebhooks:
		return "Manage Webhooks"
	case discordgo.PermissionManageEmojis:
		return "Manage Emojis"
	default:
		return fmt.Sprintf("Permission %d", perm)
	}
}
