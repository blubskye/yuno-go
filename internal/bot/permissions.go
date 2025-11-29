package bot

import (
	"log"
	"sort"

	"github.com/bwmarrin/discordgo"
)

// PermissionChecker handles permission and hierarchy validation
type PermissionChecker struct {
	session *discordgo.Session
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(s *discordgo.Session) *PermissionChecker {
	return &PermissionChecker{session: s}
}

// HasPermission checks if a user has a specific permission in a guild
func (pc *PermissionChecker) HasPermission(guildID, userID string, permission int64) bool {
	member, err := pc.session.GuildMember(guildID, userID)
	if err != nil {
		DebugLog("Failed to get member %s in guild %s: %v", userID, guildID, err)
		return false
	}

	guild, err := pc.session.Guild(guildID)
	if err != nil {
		DebugLog("Failed to get guild %s: %v", guildID, err)
		return false
	}

	// Check if user is guild owner
	if guild.OwnerID == userID {
		return true
	}

	// Check permissions in member's roles
	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				// Administrator permission bypasses all
				if role.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
					return true
				}
				// Check specific permission
				if role.Permissions&permission == permission {
					return true
				}
			}
		}
	}

	return false
}

// GetHighestRolePosition returns the position of the highest role a user has
func (pc *PermissionChecker) GetHighestRolePosition(guildID, userID string) int {
	member, err := pc.session.GuildMember(guildID, userID)
	if err != nil {
		DebugLog("Failed to get member %s: %v", userID, err)
		return -1
	}

	guild, err := pc.session.Guild(guildID)
	if err != nil {
		DebugLog("Failed to get guild %s: %v", guildID, err)
		return -1
	}

	// Owner has infinite position
	if guild.OwnerID == userID {
		return 999999
	}

	highestPosition := 0
	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID && role.Position > highestPosition {
				highestPosition = role.Position
			}
		}
	}

	return highestPosition
}

// IsHigherRank checks if user1 is higher rank than user2
func (pc *PermissionChecker) IsHigherRank(guildID, user1ID, user2ID string) bool {
	pos1 := pc.GetHighestRolePosition(guildID, user1ID)
	pos2 := pc.GetHighestRolePosition(guildID, user2ID)

	DebugLog("Hierarchy check: user1=%s (pos=%d) vs user2=%s (pos=%d)", user1ID, pos1, user2ID, pos2)

	return pos1 > pos2
}

// IsSameRank checks if two users have the same highest role position
func (pc *PermissionChecker) IsSameRank(guildID, user1ID, user2ID string) bool {
	pos1 := pc.GetHighestRolePosition(guildID, user1ID)
	pos2 := pc.GetHighestRolePosition(guildID, user2ID)

	return pos1 == pos2 && pos1 > 0
}

// IsOwner checks if the user is the guild owner
func (pc *PermissionChecker) IsOwner(guildID, userID string) bool {
	guild, err := pc.session.Guild(guildID)
	if err != nil {
		return false
	}
	return guild.OwnerID == userID
}

// IsBotOwner checks if user is a bot owner
func (pc *PermissionChecker) IsBotOwner(userID string) bool {
	for _, ownerID := range Global.Bot.OwnerIDs {
		if ownerID == userID {
			return true
		}
	}
	return false
}

// HasExemptRole checks if a user has any of the exempt roles
func (pc *PermissionChecker) HasExemptRole(guildID, userID string, exemptRoleIDs []string) bool {
	if len(exemptRoleIDs) == 0 {
		return false
	}

	member, err := pc.session.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	for _, memberRoleID := range member.Roles {
		for _, exemptRoleID := range exemptRoleIDs {
			if memberRoleID == exemptRoleID {
				return true
			}
		}
	}

	return false
}

// CanModerate checks if moderator can take action against target
// Returns: canModerate, reason
func (pc *PermissionChecker) CanModerate(guildID, moderatorID, targetID string) (bool, string) {
	// Bot owners can do anything
	if pc.IsBotOwner(moderatorID) {
		return true, ""
	}

	// Guild owner can moderate anyone except bot owners
	if pc.IsOwner(guildID, moderatorID) {
		if pc.IsBotOwner(targetID) {
			return false, "Cannot moderate bot owner"
		}
		return true, ""
	}

	// Can't moderate bot owners
	if pc.IsBotOwner(targetID) {
		return false, "Cannot moderate bot owner"
	}

	// Can't moderate guild owner
	if pc.IsOwner(guildID, targetID) {
		return false, "Cannot moderate server owner"
	}

	// Check if moderator has ban permissions
	if !pc.HasPermission(guildID, moderatorID, discordgo.PermissionBanMembers) {
		return false, "You don't have ban permissions"
	}

	// Check hierarchy
	if !pc.IsHigherRank(guildID, moderatorID, targetID) {
		// Check if same rank moderation is allowed
		if pc.IsSameRank(guildID, moderatorID, targetID) && Global.SpamFilter.AllowSameRoleModeration {
			return true, ""
		}
		return false, "Target has equal or higher role"
	}

	return true, ""
}

// GetRolesSorted returns roles sorted by position (highest first)
func (pc *PermissionChecker) GetRolesSorted(guildID string) ([]*discordgo.Role, error) {
	guild, err := pc.session.Guild(guildID)
	if err != nil {
		return nil, err
	}

	roles := make([]*discordgo.Role, len(guild.Roles))
	copy(roles, guild.Roles)

	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})

	return roles, nil
}

// AutoBanViolator bans a user for violating moderation rules
func (pc *PermissionChecker) AutoBanViolator(guildID, userID, reason string) error {
	DebugLog("Auto-banning user %s in guild %s: %s", userID, guildID, reason)

	err := pc.session.GuildBanCreateWithReason(guildID, userID, reason, 1)
	if err != nil {
		log.Printf("❌ Failed to auto-ban %s: %v", userID, err)
		return err
	}

	log.Printf("🔨 Auto-banned user %s: %s", userID, reason)
	return nil
}
