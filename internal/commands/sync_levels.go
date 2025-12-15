package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SyncLevelsCommand syncs member levels based on role names
type SyncLevelsCommand struct{}

func (c *SyncLevelsCommand) Name() string        { return "sync-levels" }
func (c *SyncLevelsCommand) Aliases() []string   { return []string{"synclvl", "syncxp"} }
func (c *SyncLevelsCommand) Description() string { return "Sync member XP from role names (Lvl X+)" }
func (c *SyncLevelsCommand) Usage() string       { return "sync-levels" }
func (c *SyncLevelsCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *SyncLevelsCommand) MasterOnly() bool { return false }

func (c *SyncLevelsCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	// Send initial status message
	statusMsg, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, &discordgo.MessageEmbed{
		Title:       "🔄 Syncing Levels...",
		Description: "Fetching all server members...",
		Color:       0xFFAA00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "This may take a while for large servers",
		},
	})
	if err != nil {
		return err
	}

	// Get all guild members
	members := []*discordgo.Member{}
	after := ""
	
	for {
		chunk, err := ctx.Session.GuildMembers(ctx.Message.GuildID, after, 1000)
		if err != nil {
			ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
				Title:       "❌ Sync Failed",
				Description: fmt.Sprintf("Failed to fetch members: %v", err),
				Color:       0xFF0000,
			})
			return err
		}
		
		if len(chunk) == 0 {
			break
		}
		
		members = append(members, chunk...)
		after = chunk[len(chunk)-1].User.ID
		
		// Update status every 1000 members
		if len(members)%1000 == 0 {
			ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
				Title:       "🔄 Syncing Levels...",
				Description: fmt.Sprintf("Fetched %d members so far...", len(members)),
				Color:       0xFFAA00,
			})
		}
	}

	// Get all guild roles
	guild, err := ctx.Session.Guild(ctx.Message.GuildID)
	if err != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
			Title:       "❌ Sync Failed",
			Description: fmt.Sprintf("Failed to fetch guild: %v", err),
			Color:       0xFF0000,
		})
		return err
	}

	// Create role ID to level map
	roleToLevel := make(map[string]int)
	levelRegex := regexp.MustCompile(`\(Lvl\s*(\d+)\+?\)`)
	
	for _, role := range guild.Roles {
		matches := levelRegex.FindStringSubmatch(role.Name)
		if len(matches) >= 2 {
			var level int
			fmt.Sscanf(matches[1], "%d", &level)
			roleToLevel[role.ID] = level
		}
	}

	if len(roleToLevel) == 0 {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, &discordgo.MessageEmbed{
			Title:       "⚠️ No Level Roles Found",
			Description: "No roles with format `(Lvl X+)` were found in this server.",
			Color:       0xFFAA00,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Example role name: Member (Lvl 5+)",
			},
		})
		return nil
	}

	// Process members
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()
	
	processed := 0
	updated := 0
	skipped := 0
	errors := 0
	
	startTime := time.Now()
	lastUpdate := time.Now()

	for i, member := range members {
		// Skip bots
		if member.User.Bot {
			skipped++
			processed++
			continue
		}

		// Find highest level role
		highestLevel := 0
		for _, roleID := range member.Roles {
			if level, exists := roleToLevel[roleID]; exists {
				highestLevel = max(highestLevel, level)
			}
		}

		// Skip if no level role
		if highestLevel == 0 {
			skipped++
			processed++
			continue
		}

		// Calculate XP for this level (start of level)
		xp := 0
		for lvl := 0; lvl < highestLevel; lvl++ {
			xp += 5*lvl*lvl + 50*lvl + 100
		}

		// Update database
		_, err := db.Exec(`
			INSERT OR REPLACE INTO glevel (guild_id, user_id, exp, level, enabled)
			VALUES (?, ?, ?, ?, 'enabled')`,
			ctx.Message.GuildID, member.User.ID, xp, highestLevel)

		if err != nil {
			errors++
		} else {
			updated++
		}
		
		processed++

		// Update progress every 2 seconds or every 100 members
		if time.Since(lastUpdate) >= 2*time.Second || (i+1)%100 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(processed) / elapsed.Seconds()
			remaining := float64(len(members)-processed) / rate

			embed := &discordgo.MessageEmbed{
				Title:       "🔄 Syncing Levels...",
				Description: fmt.Sprintf("Processing members with level roles..."),
				Color:       0xFFAA00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Progress",
						Value:  fmt.Sprintf("%d / %d members (%.1f%%)", processed, len(members), float64(processed)/float64(len(members))*100),
						Inline: true,
					},
					{
						Name:   "Updated",
						Value:  fmt.Sprintf("%d members", updated),
						Inline: true,
					},
					{
						Name:   "Skipped",
						Value:  fmt.Sprintf("%d (bots/no role)", skipped),
						Inline: true,
					},
					{
						Name:   "Speed",
						Value:  fmt.Sprintf("%.1f members/sec", rate),
						Inline: true,
					},
					{
						Name:   "ETA",
						Value:  fmt.Sprintf("~%.0f seconds", remaining),
						Inline: true,
					},
				},
			}

			if errors > 0 {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   "Errors",
					Value:  fmt.Sprintf("%d errors", errors),
					Inline: true,
				})
			}

			ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, embed)
			lastUpdate = time.Now()
		}
	}

	// Final summary
	elapsed := time.Since(startTime)
	
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Level Sync Complete!",
		Description: fmt.Sprintf("Successfully synced levels from role names"),
		Color:       0x43CC24,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Total Members",
				Value:  strconv.Itoa(len(members)),
				Inline: true,
			},
			{
				Name:   "Updated",
				Value:  fmt.Sprintf("%d members", updated),
				Inline: true,
			},
			{
				Name:   "Skipped",
				Value:  fmt.Sprintf("%d (bots/no role)", skipped),
				Inline: true,
			},
			{
				Name:   "Time Taken",
				Value:  fmt.Sprintf("%.1f seconds", elapsed.Seconds()),
				Inline: true,
			},
			{
				Name:   "Roles Found",
				Value:  fmt.Sprintf("%d level roles", len(roleToLevel)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Requested by %s", ctx.Message.Author.Username),
		},
	}

	if errors > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Errors",
			Value:  fmt.Sprintf("%d errors occurred", errors),
			Inline: true,
		})
		embed.Color = 0xFFAA00
	}

	// Show detected roles
	roleList := []string{}
	for roleID, level := range roleToLevel {
		role, err := ctx.Session.State.Role(ctx.Message.GuildID, roleID)
		if err == nil {
			roleList = append(roleList, fmt.Sprintf("• %s → Level %d", role.Name, level))
		}
	}
	
	if len(roleList) > 0 {
		roleListStr := strings.Join(roleList, "\n")
		if len(roleListStr) > 1024 {
			roleListStr = roleListStr[:1021] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Detected Level Roles",
			Value:  roleListStr,
			Inline: false,
		})
	}

	ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, statusMsg.ID, embed)
	return nil
}

// PreviewLevelsCommand shows what would be synced without doing it
type PreviewLevelsCommand struct{}

func (c *PreviewLevelsCommand) Name() string        { return "preview-levels" }
func (c *PreviewLevelsCommand) Aliases() []string   { return []string{"previewlvl"} }
func (c *PreviewLevelsCommand) Description() string { return "Preview level sync without changing data" }
func (c *PreviewLevelsCommand) Usage() string       { return "preview-levels" }
func (c *PreviewLevelsCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageRoles}
}
func (c *PreviewLevelsCommand) MasterOnly() bool { return false }

func (c *PreviewLevelsCommand) Execute(ctx *Context) error {
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
	roleToLevel := make(map[string]int)
	levelRegex := regexp.MustCompile(`\(Lvl\s*(\d+)\+?\)`)
	
	roleList := []string{}
	for _, role := range guild.Roles {
		matches := levelRegex.FindStringSubmatch(role.Name)
		if len(matches) >= 2 {
			var level int
			fmt.Sscanf(matches[1], "%d", &level)
			roleToLevel[role.ID] = level
			roleList = append(roleList, fmt.Sprintf("• **%s** → Level %d", role.Name, level))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Level Role Preview",
		Color: 0xFF51FF,
	}

	if len(roleList) == 0 {
		embed.Description = "⚠️ No roles found with format `(Lvl X+)`\n\nExample: `Member (Lvl 5+)`"
	} else {
		embed.Description = fmt.Sprintf("Found **%d** roles with level indicators:\n\n%s\n\nUse `sync-levels` to apply these levels to all members.",
			len(roleList), strings.Join(roleList, "\n"))
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
