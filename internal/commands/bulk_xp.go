package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// XP calculation constants
const levelDivisor = 50

func calcLevel(exp int64) int {
	return int((math.Sqrt(1+8*float64(exp)/levelDivisor) - 1) / 2)
}

func calcExpForLevel(level int) int64 {
	return int64(levelDivisor * level * (level + 1) / 2)
}

// XPDatabase interface for XP operations
type XPDatabase interface {
	GetUserXP(guildID, userID string) (exp int64, level int, err error)
	SetUserXP(guildID, userID string, exp int64, level int) error
	AddUserXP(guildID, userID string, amount int64) error
	GetGuildRanks(guildID string) ([]GuildRank, error)
}

// GuildRank represents a level-role mapping
type GuildRank struct {
	RoleID string
	Level  int
}

// MassAddXPCommand adds XP to all members with a role
type MassAddXPCommand struct {
	DB XPDatabase
}

func (c *MassAddXPCommand) Name() string        { return "mass-addxp" }
func (c *MassAddXPCommand) Aliases() []string   { return []string{"massaddxp", "bulkaddxp"} }
func (c *MassAddXPCommand) Description() string { return "Add XP to all members with a specific role" }
func (c *MassAddXPCommand) Usage() string       { return "mass-addxp <@role> <amount>" }
func (c *MassAddXPCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *MassAddXPCommand) MasterOnly() bool { return false }

func (c *MassAddXPCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("Usage: `%smass-addxp @Role 500`", ctx.GetPrefix()))
		return nil
	}

	// Get role
	var roleID string
	if len(ctx.Message.MentionRoles) > 0 {
		roleID = ctx.Message.MentionRoles[0]
	} else {
		// Try to find role by name
		roleName := ctx.Args[0]
		guild, err := ctx.Session.Guild(ctx.Message.GuildID)
		if err != nil {
			return err
		}
		for _, role := range guild.Roles {
			if strings.EqualFold(role.Name, roleName) {
				roleID = role.ID
				break
			}
		}
	}

	if roleID == "" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Role not found.")
		return nil
	}

	// Get amount
	amountStr := ctx.Args[len(ctx.Args)-1]
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Amount must be a positive number.")
		return nil
	}

	if amount > 100000 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Maximum 100,000 XP per operation.")
		return nil
	}

	// Get role name
	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error finding role.")
		return err
	}

	// Send initial message
	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("Adding %d XP to members with %s...", amount, role.Name))

	// Get guild members
	members, err := ctx.Session.GuildMembers(ctx.Message.GuildID, "", 1000)
	if err != nil {
		return err
	}

	updated := 0
	levelUps := 0

	for _, member := range members {
		if member.User.Bot {
			continue
		}

		// Check if member has the role
		hasRole := false
		for _, r := range member.Roles {
			if r == roleID {
				hasRole = true
				break
			}
		}
		if !hasRole {
			continue
		}

		// Get current XP
		exp, oldLevel, _ := c.DB.GetUserXP(ctx.Message.GuildID, member.User.ID)

		// Add XP
		newExp := exp + amount
		newLevel := calcLevel(newExp)

		if err := c.DB.SetUserXP(ctx.Message.GuildID, member.User.ID, newExp, newLevel); err != nil {
			continue
		}

		updated++
		if newLevel > oldLevel {
			levelUps++
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "✅ Bulk XP Added",
		Color: 0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Role", Value: role.Mention(), Inline: true},
			{Name: "XP Added", Value: fmt.Sprintf("+%d", amount), Inline: true},
			{Name: "Members Updated", Value: strconv.Itoa(updated), Inline: true},
			{Name: "Level Ups", Value: strconv.Itoa(levelUps), Inline: true},
		},
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}

// MassSetXPCommand sets XP for all members with a role
type MassSetXPCommand struct {
	DB XPDatabase
}

func (c *MassSetXPCommand) Name() string        { return "mass-setxp" }
func (c *MassSetXPCommand) Aliases() []string   { return []string{"masssetxp", "bulksetxp"} }
func (c *MassSetXPCommand) Description() string { return "Set XP for all members with a specific role" }
func (c *MassSetXPCommand) Usage() string       { return "mass-setxp <@role> <amount>" }
func (c *MassSetXPCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *MassSetXPCommand) MasterOnly() bool { return false }

func (c *MassSetXPCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("Usage: `%smass-setxp @Role 1000`", ctx.GetPrefix()))
		return nil
	}

	// Get role
	var roleID string
	if len(ctx.Message.MentionRoles) > 0 {
		roleID = ctx.Message.MentionRoles[0]
	} else {
		roleName := ctx.Args[0]
		guild, err := ctx.Session.Guild(ctx.Message.GuildID)
		if err != nil {
			return err
		}
		for _, role := range guild.Roles {
			if strings.EqualFold(role.Name, roleName) {
				roleID = role.ID
				break
			}
		}
	}

	if roleID == "" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Role not found.")
		return nil
	}

	// Get amount
	amountStr := ctx.Args[len(ctx.Args)-1]
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount < 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Amount cannot be negative.")
		return nil
	}

	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		return err
	}

	newLevel := calcLevel(amount)

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("Setting XP to %d for members with %s...", amount, role.Name))

	members, err := ctx.Session.GuildMembers(ctx.Message.GuildID, "", 1000)
	if err != nil {
		return err
	}

	updated := 0

	for _, member := range members {
		if member.User.Bot {
			continue
		}

		hasRole := false
		for _, r := range member.Roles {
			if r == roleID {
				hasRole = true
				break
			}
		}
		if !hasRole {
			continue
		}

		if err := c.DB.SetUserXP(ctx.Message.GuildID, member.User.ID, amount, newLevel); err != nil {
			continue
		}
		updated++
	}

	embed := &discordgo.MessageEmbed{
		Title: "✅ Bulk XP Set",
		Color: 0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Role", Value: role.Mention(), Inline: true},
			{Name: "XP Set To", Value: strconv.FormatInt(amount, 10), Inline: true},
			{Name: "Level", Value: strconv.Itoa(newLevel), Inline: true},
			{Name: "Members Updated", Value: strconv.Itoa(updated), Inline: true},
		},
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}

// MassLevelUpCommand increases level for all members with a role
type MassLevelUpCommand struct {
	DB XPDatabase
}

func (c *MassLevelUpCommand) Name() string        { return "mass-levelup" }
func (c *MassLevelUpCommand) Aliases() []string   { return []string{"masslevelup", "bulklevelup"} }
func (c *MassLevelUpCommand) Description() string { return "Increase level for all members with a role" }
func (c *MassLevelUpCommand) Usage() string       { return "mass-levelup <@role> [levels]" }
func (c *MassLevelUpCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *MassLevelUpCommand) MasterOnly() bool { return false }

func (c *MassLevelUpCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) < 1 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("Usage: `%smass-levelup @Role [levels]`", ctx.GetPrefix()))
		return nil
	}

	// Get role
	var roleID string
	if len(ctx.Message.MentionRoles) > 0 {
		roleID = ctx.Message.MentionRoles[0]
	} else {
		roleName := ctx.Args[0]
		guild, err := ctx.Session.Guild(ctx.Message.GuildID)
		if err != nil {
			return err
		}
		for _, role := range guild.Roles {
			if strings.EqualFold(role.Name, roleName) {
				roleID = role.ID
				break
			}
		}
	}

	if roleID == "" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Role not found.")
		return nil
	}

	// Get levels (default 1)
	levels := 1
	if len(ctx.Args) > 1 {
		if l, err := strconv.Atoi(ctx.Args[len(ctx.Args)-1]); err == nil && l > 0 {
			levels = l
		}
	}

	if levels > 100 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Maximum 100 levels per operation.")
		return nil
	}

	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		return err
	}

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("Increasing level by %d for members with %s...", levels, role.Name))

	members, err := ctx.Session.GuildMembers(ctx.Message.GuildID, "", 1000)
	if err != nil {
		return err
	}

	updated := 0

	for _, member := range members {
		if member.User.Bot {
			continue
		}

		hasRole := false
		for _, r := range member.Roles {
			if r == roleID {
				hasRole = true
				break
			}
		}
		if !hasRole {
			continue
		}

		_, oldLevel, _ := c.DB.GetUserXP(ctx.Message.GuildID, member.User.ID)
		newLevel := oldLevel + levels
		newExp := calcExpForLevel(newLevel)

		if err := c.DB.SetUserXP(ctx.Message.GuildID, member.User.ID, newExp, newLevel); err != nil {
			continue
		}
		updated++
	}

	embed := &discordgo.MessageEmbed{
		Title: "✅ Bulk Level Up",
		Color: 0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Role", Value: role.Mention(), Inline: true},
			{Name: "Levels Added", Value: fmt.Sprintf("+%d", levels), Inline: true},
			{Name: "Members Updated", Value: strconv.Itoa(updated), Inline: true},
		},
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}

// SyncXPFromRolesCommand syncs XP from roles
type SyncXPFromRolesCommand struct {
	DB XPDatabase
}

func (c *SyncXPFromRolesCommand) Name() string        { return "sync-xp-from-roles" }
func (c *SyncXPFromRolesCommand) Aliases() []string   { return []string{"syncxpfromroles", "rolesync"} }
func (c *SyncXPFromRolesCommand) Description() string { return "Sync member XP based on their highest level role" }
func (c *SyncXPFromRolesCommand) Usage() string       { return "sync-xp-from-roles" }
func (c *SyncXPFromRolesCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionAdministrator}
}
func (c *SyncXPFromRolesCommand) MasterOnly() bool { return false }

func (c *SyncXPFromRolesCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Syncing XP from roles...")

	// Get rank mappings
	ranks, err := c.DB.GetGuildRanks(ctx.Message.GuildID)
	if err != nil || len(ranks) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("No level roles configured. Use `%sranks add` first.", ctx.GetPrefix()))
		return nil
	}

	members, err := ctx.Session.GuildMembers(ctx.Message.GuildID, "", 1000)
	if err != nil {
		return err
	}

	updated := 0

	for _, member := range members {
		if member.User.Bot {
			continue
		}

		// Find highest level role the member has
		highestLevel := 0
		for _, rank := range ranks {
			for _, memberRole := range member.Roles {
				if memberRole == rank.RoleID && rank.Level > highestLevel {
					highestLevel = rank.Level
				}
			}
		}

		if highestLevel > 0 {
			exp := calcExpForLevel(highestLevel)
			if err := c.DB.SetUserXP(ctx.Message.GuildID, member.User.ID, exp, highestLevel); err == nil {
				updated++
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ XP Synced from Roles",
		Description: fmt.Sprintf("Updated %d members based on their level roles.", updated),
		Color:       0x00FF00,
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}
