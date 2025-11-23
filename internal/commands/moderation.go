package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// BanCommand bans users
type BanCommand struct{}

func (c *BanCommand) Name() string        { return "ban" }
func (c *BanCommand) Aliases() []string   { return []string{"bean", "banne"} }
func (c *BanCommand) Description() string { return "Ban users from the server" }
func (c *BanCommand) Usage() string       { return "ban <@user|id> [more...] | reason" }
func (c *BanCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *BanCommand) MasterOnly() bool { return false }

func (c *BanCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Usage: `ban <@user|id> [more...] | reason`")
		return nil
	}

	// Parse reason and targets
	fullArgs := strings.Join(ctx.Args, " ")
	parts := strings.Split(fullArgs, "|")
	
	reason := "Banned by " + ctx.Message.Author.Username
	targets := parts[0]
	if len(parts) > 1 {
		reason = strings.TrimSpace(parts[1]) + " / " + reason
	}

	// Collect users to ban
	var userIDs []string
	
	// Add mentioned users
	for _, user := range ctx.Message.Mentions {
		userIDs = append(userIDs, user.ID)
	}

	// Parse remaining arguments for IDs
	targetParts := strings.Fields(targets)
	for _, part := range targetParts {
		// Skip mentions (already processed)
		if strings.HasPrefix(part, "<@") {
			continue
		}
		
		// Check if it looks like a user ID (numeric)
		if len(part) >= 17 && len(part) <= 19 {
			// Verify it's a valid user
			_, err := ctx.Session.User(part)
			if err == nil {
				userIDs = append(userIDs, part)
			}
		}
	}

	if len(userIDs) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ No valid users to ban")
		return nil
	}

	// Ban each user
	successCount := 0
	failCount := 0
	
	for _, userID := range userIDs {
		// Check if user is a master user
		// (You'd need to implement this check)
		
		err := ctx.Session.GuildBanCreateWithReason(
			ctx.Message.GuildID,
			userID,
			reason,
			1, // Delete 1 day of messages
		)

		if err != nil {
			failCount++
			
			// Send error embed
			embed := &discordgo.MessageEmbed{
				Title:       "❌ Ban failed",
				Description: fmt.Sprintf("Failed to ban <@%s>: %s", userID, err.Error()),
				Color:       0xFF0000,
			}
			ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		} else {
			successCount++
			
			// Get user info
			user, _ := ctx.Session.User(userID)
			username := userID
			if user != nil {
				username = user.Username + "#" + user.Discriminator
			}
			
			// Send success embed
			embed := &discordgo.MessageEmbed{
				Title:       "✅ Ban successful",
				Description: fmt.Sprintf("User %s has been successfully banned.", username),
				Color:       0x43CC24,
			}
			ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		}
	}

	return nil
}

// KickCommand kicks users
type KickCommand struct{}

func (c *KickCommand) Name() string        { return "kick" }
func (c *KickCommand) Aliases() []string   { return []string{} }
func (c *KickCommand) Description() string { return "Kick users from the server" }
func (c *KickCommand) Usage() string       { return "kick <@user|id> [more...] | reason" }
func (c *KickCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionKickMembers}
}
func (c *KickCommand) MasterOnly() bool { return false }

func (c *KickCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ Usage: `kick <@user|id> [more...] | reason`")
		return nil
	}

	// Parse reason and targets
	fullArgs := strings.Join(ctx.Args, " ")
	parts := strings.Split(fullArgs, "|")
	
	reason := "Kicked by " + ctx.Message.Author.Username
	targets := parts[0]
	if len(parts) > 1 {
		reason = strings.TrimSpace(parts[1]) + " / " + reason
	}

	// Collect users to kick
	var userIDs []string
	
	// Add mentioned users
	for _, user := range ctx.Message.Mentions {
		userIDs = append(userIDs, user.ID)
	}

	// Parse IDs
	targetParts := strings.Fields(targets)
	for _, part := range targetParts {
		if strings.HasPrefix(part, "<@") {
			continue
		}
		
		if len(part) >= 17 && len(part) <= 19 {
			member, err := ctx.Session.GuildMember(ctx.Message.GuildID, part)
			if err == nil && member != nil {
				userIDs = append(userIDs, part)
			}
		}
	}

	if len(userIDs) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "❌ No valid users to kick")
		return nil
	}

	// Kick each user
	for _, userID := range userIDs {
		// Check if kicking self
		if userID == ctx.Message.Author.ID {
			embed := &discordgo.MessageEmbed{
				Title:       "❌ Kick failed",
				Description: "You can also leave the server instead of kicking yourself ;)",
				Color:       0xFF0000,
			}
			ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
			continue
		}

		err := ctx.Session.GuildMemberDeleteWithReason(
			ctx.Message.GuildID,
			userID,
			reason,
		)

		if err != nil {
			embed := &discordgo.MessageEmbed{
				Title:       "❌ Kick failed",
				Description: fmt.Sprintf("Failed to kick <@%s>: %s", userID, err.Error()),
				Color:       0xFF0000,
			}
			ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		} else {
			member, _ := ctx.Session.GuildMember(ctx.Message.GuildID, userID)
			username := userID
			if member != nil {
				username = member.User.Username
			}
			
			embed := &discordgo.MessageEmbed{
				Title:       "✅ Kick successful",
				Description: fmt.Sprintf("User %s has been successfully kicked.", username),
				Color:       0x43CC24,
			}
			ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		}
	}

	return nil
}