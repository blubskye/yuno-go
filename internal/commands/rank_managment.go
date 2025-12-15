package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SyncRanksCommand syncs rank roles from role names to database
type SyncRanksCommand struct{}

func (c *SyncRanksCommand) Name() string        { return "sync-ranks" }
func (c *SyncRanksCommand) Aliases() []string   { return []string{"syncranks", "syncroles"} }
func (c *SyncRanksCommand) Description() string { return "Sync rank roles from role names (Lvl X+) to auto-assign" }
func (c *SyncRanksCommand) Usage() string       { return "sync-ranks" }
func (c *SyncRanksCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *SyncRanksCommand) MasterOnly() bool { return false }

func (c *SyncRanksCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	// Get all guild roles
	guild, err := ctx.Session.Guild(ctx.Message.GuildID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to fetch guild: %v", err))
		return err
	}

	// Find roles with level format
	levelRegex := regexp.MustCompile(`\(Lvl\s*(\d+)\+?\)`)
	
	found := []struct {
		RoleID string
		Name   string
		Level  int
	}{}

	for _, role := range guild.Roles {
		matches := levelRegex.FindStringSubmatch(role.Name)
		if len(matches) >= 2 {
			var level int
			fmt.Sscanf(matches[1], "%d", &level)
			found = append(found, struct {
				RoleID string
				Name   string
				Level  int
			}{role.ID, role.Name, level})
		}
	}

	if len(found) == 0 {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, &discordgo.MessageEmbed{
			Title:       "⚠️ No Level Roles Found",
			Description: "No roles with format `(Lvl X+)` were found in this server.",
			Color:       0xFFAA00,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Example role name: Member (Lvl 5+)",
			},
		})
		return nil
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Clear existing ranks for this guild
	_, err = db.Exec(`DELETE FROM ranks WHERE guild_id = ?`, ctx.Message.GuildID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to clear old ranks: %v", err))
		return err
	}

	// Insert new ranks
	success := 0
	failures := 0
	roleList := []string{}

	for _, r := range found {
		_, err := db.Exec(`
			INSERT INTO ranks (guild_id, role_id, level)
			VALUES (?, ?, ?)`,
			ctx.Message.GuildID, r.RoleID, r.Level)

		if err != nil {
			failures++
		} else {
			success++
			roleList = append(roleList, fmt.Sprintf("• <@&%s> → Level %d", r.RoleID, r.Level))
		}
	}

	// Create summary embed
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Rank Roles Synced",
		Description: fmt.Sprintf("Successfully configured **%d** rank roles for auto-assignment.\n\nMembers will automatically receive these roles when they reach the required level.", success),
		Color:       0x43CC24,
	}

	if len(roleList) > 0 {
		roleListStr := strings.Join(roleList, "\n")
		if len(roleListStr) > 1024 {
			roleListStr = roleListStr[:1021] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Configured Roles",
			Value:  roleListStr,
			Inline: false,
		})
	}

	if failures > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Failures",
			Value:  fmt.Sprintf("%d roles failed to sync", failures),
			Inline: true,
		})
		embed.Color = 0xFFAA00
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Use 'list-ranks' to view all configured ranks",
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// AddRankCommand manually adds a rank role
type AddRankCommand struct{}

func (c *AddRankCommand) Name() string        { return "add-rank" }
func (c *AddRankCommand) Aliases() []string   { return []string{"addrank", "rankrole"} }
func (c *AddRankCommand) Description() string { return "Manually add a rank role for auto-assignment" }
func (c *AddRankCommand) Usage() string       { return "add-rank @role <level>" }
func (c *AddRankCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *AddRankCommand) MasterOnly() bool { return false }

func (c *AddRankCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Message.MentionRoles) == 0 || len(ctx.Args) < 2 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `add-rank @role <level>`\nExample: `add-rank @Member 5`")
		return nil
	}

	roleID := ctx.Message.MentionRoles[0]
	level, err := strconv.Atoi(ctx.Args[1])
	if err != nil || level < 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Level must be a positive number")
		return nil
	}

	// Get role info
	role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Failed to fetch role information")
		return err
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Insert or replace rank
	_, err = db.Exec(`
		INSERT OR REPLACE INTO ranks (guild_id, role_id, level)
		VALUES (?, ?, ?)`,
		ctx.Message.GuildID, roleID, level)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to add rank: %v", err))
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Rank Role Added",
		Description: fmt.Sprintf("Members who reach **Level %d** will automatically receive the <@&%s> role.", level, roleID),
		Color:       0x43CC24,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Role",
				Value:  role.Name,
				Inline: true,
			},
			{
				Name:   "Required Level",
				Value:  strconv.Itoa(level),
				Inline: true,
			},
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// RemoveRankCommand removes a rank role
type RemoveRankCommand struct{}

func (c *RemoveRankCommand) Name() string        { return "remove-rank" }
func (c *RemoveRankCommand) Aliases() []string   { return []string{"delrank", "deleterank"} }
func (c *RemoveRankCommand) Description() string { return "Remove a rank role from auto-assignment" }
func (c *RemoveRankCommand) Usage() string       { return "remove-rank @role" }
func (c *RemoveRankCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *RemoveRankCommand) MasterOnly() bool { return false }

func (c *RemoveRankCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Message.MentionRoles) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `remove-rank @role`")
		return nil
	}

	roleID := ctx.Message.MentionRoles[0]

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Delete rank
	_, err := db.Exec(`
		DELETE FROM ranks 
		WHERE guild_id = ? AND role_id = ?`,
		ctx.Message.GuildID, roleID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to remove rank: %v", err))
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Rank Role Removed",
		Description: fmt.Sprintf("<@&%s> will no longer be automatically assigned.", roleID),
		Color:       0x43CC24,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ListRanksCommand lists all rank roles
type ListRanksCommand struct{}

func (c *ListRanksCommand) Name() string        { return "list-ranks" }
func (c *ListRanksCommand) Aliases() []string   { return []string{"ranks", "listroles"} }
func (c *ListRanksCommand) Description() string { return "List all configured rank roles" }
func (c *ListRanksCommand) Usage() string       { return "list-ranks" }
func (c *ListRanksCommand) RequiredPermissions() []int64 { return nil }
func (c *ListRanksCommand) MasterOnly() bool    { return false }

func (c *ListRanksCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Query ranks
	rows, err := db.(interface {
		Query(query string, args ...interface{}) (interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
		}, error)
	}).Query(`
		SELECT role_id, level 
		FROM ranks 
		WHERE guild_id = ? 
		ORDER BY level ASC`,
		ctx.Message.GuildID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to fetch ranks: %v", err))
		return err
	}
	defer rows.Close()

	embed := &discordgo.MessageEmbed{
		Title:       "Rank Roles Configuration",
		Description: "Members automatically receive these roles when reaching the required level.",
		Color:       0xFF51FF,
		Fields:      []*discordgo.MessageEmbedField{},
	}

	count := 0
	roleList := []string{}
	
	for rows.Next() {
		var roleID string
		var level int

		if err := rows.Scan(&roleID, &level); err != nil {
			continue
		}

		// Try to get role name
		role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
		roleName := roleID
		if err == nil {
			roleName = role.Name
		}

		roleList = append(roleList, fmt.Sprintf("**Level %d** → <@&%s> (%s)", level, roleID, roleName))
		count++
	}

	if count == 0 {
		embed.Description = "No rank roles configured. Use `sync-ranks` to auto-detect from role names, or `add-rank` to add manually."
		embed.Color = 0xFFAA00
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("📊 %d Rank Role(s)", count),
			Value: strings.Join(roleList, "\n"),
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Use 'sync-ranks' to auto-detect | 'add-rank' to add manually",
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ApplyRanksCommand applies rank roles to all eligible members
type ApplyRanksCommand struct{}

func (c *ApplyRanksCommand) Name() string        { return "apply-ranks" }
func (c *ApplyRanksCommand) Aliases() []string   { return []string{"applyranks"} }
func (c *ApplyRanksCommand) Description() string { return "Apply rank roles to all eligible members based on their current levels" }
func (c *ApplyRanksCommand) Usage() string       { return "apply-ranks" }
func (c *ApplyRanksCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *ApplyRanksCommand) MasterOnly() bool { return false }

func (c *ApplyRanksCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	statusMsg, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, &discordgo.MessageEmbed{
		Title:       "🔄 Applying Ranks...",
		Description: "Fetching rank configuration and member data...",
		Color:       0xFFAA00,
	})
	if err != nil {
		return err
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Get all rank roles
	rows, err := db.(interface {
		Query(query string, args ...interface{}) (interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
		}, error)
	}).Query(`
		SELECT role_id, level 
		FROM ranks 
		WHERE guild_id = ? 
		ORDER BY level ASC`,
		ctx.Message.GuildID)

	if err != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
			Title:       "❌ Failed",
			Description: fmt.Sprintf("Error fetching ranks: %v", err),
			Color:       0xFF0000,
		})
		return err
	}

	rankRoles := make(map[int]string) // level -> roleID
	for rows.Next() {
		var roleID string
		var level int
		if err := rows.Scan(&roleID, &level); err == nil {
			rankRoles[level] = roleID
		}
	}
	rows.Close()

	if len(rankRoles) == 0 {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
			Title:       "⚠️ No Ranks Configured",
			Description: "Use `sync-ranks` or `add-rank` to configure rank roles first.",
			Color:       0xFFAA00,
		})
		return nil
	}

	// Get all members with levels
	levelRows, err := db.(interface {
		Query(query string, args ...interface{}) (interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
		}, error)
	}).Query(`
		SELECT user_id, level 
		FROM glevel 
		WHERE guild_id = ?`,
		ctx.Message.GuildID)

	if err != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
			Title:       "❌ Failed",
			Description: fmt.Sprintf("Error fetching member levels: %v", err),
			Color:       0xFF0000,
		})
		return err
	}
	defer levelRows.Close()

	processed := 0
	rolesAdded := 0
	errors := 0

	for levelRows.Next() {
		var userID string
		var level int
		
		if err := levelRows.Scan(&userID, &level); err != nil {
			continue
		}

		// Find all roles this user should have
		for reqLevel, roleID := range rankRoles {
			if level >= reqLevel {
				err := ctx.Session.GuildMemberRoleAdd(ctx.Message.GuildID, userID, roleID)
				if err == nil {
					rolesAdded++
				} else {
					errors++
				}
			}
		}

		processed++

		// Update every 50 members
		if processed%50 == 0 {
			ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
				Title:       "🔄 Applying Ranks...",
				Description: fmt.Sprintf("Processed %d members...", processed),
				Color:       0xFFAA00,
			})
		}
	}

	// Final summary
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Ranks Applied",
		Description: fmt.Sprintf("Finished applying rank roles to members."),
		Color:       0x43CC24,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Members Processed",
				Value:  strconv.Itoa(processed),
				Inline: true,
			},
			{
				Name:   "Roles Added",
				Value:  strconv.Itoa(rolesAdded),
				Inline: true,
			},
			{
				Name:   "Rank Roles",
				Value:  fmt.Sprintf("%d configured", len(rankRoles)),
				Inline: true,
			},
		},
	}

	if errors > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Errors",
			Value:  fmt.Sprintf("%d errors (missing permissions?)", errors),
			Inline: true,
		})
		embed.Color = 0xFFAA00
	}

	ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, embed)
	return nil
}
