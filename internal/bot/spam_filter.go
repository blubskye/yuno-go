package bot

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SpamFilter handles message filtering and auto-moderation
type SpamFilter struct {
	bot     *Bot
	permChecker *PermissionChecker
}

// NewSpamFilter creates a new spam filter
func NewSpamFilter(b *Bot) *SpamFilter {
	return &SpamFilter{
		bot:         b,
		permChecker: NewPermissionChecker(b.Session),
	}
}

// FilterAction represents what action to take
type FilterAction string

const (
	ActionWarn   FilterAction = "warn"
	ActionBan    FilterAction = "ban"
	ActionDelete FilterAction = "delete"
)

// FilterResult contains the result of filtering a message
type FilterResult struct {
	ShouldTakeAction bool
	Action           FilterAction
	Reason           string
	RuleID           int
}

// CheckMessage checks a message against all filters
func (sf *SpamFilter) CheckMessage(m *discordgo.MessageCreate) *FilterResult {
	defer RecoverFromPanic("SpamFilter.CheckMessage")

	if m.GuildID == "" || m.Author.Bot {
		return nil
	}

	// Check @everyone and @here mentions
	if result := sf.checkMentions(m); result != nil {
		return result
	}

	// Check guild-wide regex filters
	if result := sf.checkRegexFilters(m); result != nil {
		return result
	}

	// Check channel-specific regex filters
	if result := sf.checkChannelFilters(m); result != nil {
		return result
	}

	return nil
}

// checkMentions checks for @everyone and @here mentions
func (sf *SpamFilter) checkMentions(m *discordgo.MessageCreate) *FilterResult {
	// Check if exempt role
	if sf.permChecker.HasExemptRole(m.GuildID, m.Author.ID, Global.SpamFilter.ExemptRolesFromMentionBan) {
		DebugLog("User %s has exempt role, skipping mention check", m.Author.ID)
		return nil
	}

	// Check @everyone
	if Global.SpamFilter.AutoBanOnEveryoneMention {
		if strings.Contains(m.Content, "@everyone") || m.MentionEveryone {
			DebugLog("Detected @everyone mention from %s", m.Author.ID)
			return &FilterResult{
				ShouldTakeAction: true,
				Action:           ActionBan,
				Reason:           "Unauthorized @everyone mention",
			}
		}
	}

	// Check @here
	if Global.SpamFilter.AutoBanOnHereMention {
		// Discord doesn't have a MentionHere field, so check content
		if strings.Contains(m.Content, "@here") {
			DebugLog("Detected @here mention from %s", m.Author.ID)
			return &FilterResult{
				ShouldTakeAction: true,
				Action:           ActionBan,
				Reason:           "Unauthorized @here mention",
			}
		}
	}

	return nil
}

// checkRegexFilters checks message against guild-wide regex filters
func (sf *SpamFilter) checkRegexFilters(m *discordgo.MessageCreate) *FilterResult {
	rows, err := sf.bot.DB.Query(`
		SELECT id, pattern, action, reason
		FROM regex_filters
		WHERE guild_id = ? AND enabled = 1
	`, m.GuildID)

	if err != nil {
		DebugLog("Error querying regex filters: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var pattern, action, reason string

		if err := rows.Scan(&id, &pattern, &action, &reason); err != nil {
			continue
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Invalid regex pattern in filter %d: %v", id, err)
			continue
		}

		if re.MatchString(m.Content) {
			DebugLog("Message matched regex filter %d: %s", id, pattern)
			return &FilterResult{
				ShouldTakeAction: true,
				Action:           FilterAction(action),
				Reason:           reason,
				RuleID:           id,
			}
		}
	}

	return nil
}

// checkChannelFilters checks message against channel-specific regex filters
func (sf *SpamFilter) checkChannelFilters(m *discordgo.MessageCreate) *FilterResult {
	rows, err := sf.bot.DB.Query(`
		SELECT id, pattern, action, reason
		FROM channel_regex_filters
		WHERE guild_id = ? AND channel_id = ? AND enabled = 1
	`, m.GuildID, m.ChannelID)

	if err != nil {
		DebugLog("Error querying channel filters: %v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var pattern, action, reason string

		if err := rows.Scan(&id, &pattern, &action, &reason); err != nil {
			continue
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Invalid regex pattern in channel filter %d: %v", id, err)
			continue
		}

		if re.MatchString(m.Content) {
			DebugLog("Message matched channel filter %d: %s", id, pattern)
			return &FilterResult{
				ShouldTakeAction: true,
				Action:           FilterAction(action),
				Reason:           reason,
				RuleID:           id,
			}
		}
	}

	return nil
}

// ExecuteAction executes the filter action
func (sf *SpamFilter) ExecuteAction(s *discordgo.Session, m *discordgo.MessageCreate, result *FilterResult) {
	defer RecoverFromPanic("SpamFilter.ExecuteAction")

	// Log violation
	sf.logViolation(m.GuildID, m.Author.ID, string(result.Action), result.Reason, "")

	switch result.Action {
	case ActionDelete:
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		DebugLog("Deleted message from %s: %s", m.Author.ID, result.Reason)

	case ActionWarn:
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		warning, err := s.ChannelMessageSend(m.ChannelID,
			m.Author.Mention()+" ⚠️ Warning: "+result.Reason)
		if err == nil && Global.SpamFilter.WarningLifetime > 0 {
			// Delete warning after configured time
			time.AfterFunc(time.Duration(Global.SpamFilter.WarningLifetime)*time.Second, func() {
				s.ChannelMessageDelete(m.ChannelID, warning.ID)
			})
		}
		log.Printf("⚠️  Warned user %s: %s", m.Author.Username, result.Reason)

	case ActionBan:
		// Delete the message first
		s.ChannelMessageDelete(m.ChannelID, m.ID)

		// Ban the user
		err := sf.permChecker.AutoBanViolator(m.GuildID, m.Author.ID, result.Reason)
		if err != nil {
			log.Printf("Failed to ban %s: %v", m.Author.ID, err)
		} else {
			sf.logViolation(m.GuildID, m.Author.ID, "ban", result.Reason, sf.bot.Session.State.User.ID)
		}
	}
}

// logViolation logs a spam violation to the database
func (sf *SpamFilter) logViolation(guildID, userID, violationType, reason, moderatorID string) {
	timestamp := time.Now().Format(time.RFC3339)

	_, err := sf.bot.DB.Exec(`
		INSERT INTO spam_violations
		(guild_id, user_id, violation_type, reason, timestamp, moderator_id, action_taken)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, guildID, userID, violationType, reason, timestamp, moderatorID, violationType)

	if err != nil {
		DebugLog("Failed to log violation: %v", err)
	}
}

// CheckCommandAuthorization checks if a user is authorized for a moderation command
// Returns shouldBan, reason
func (sf *SpamFilter) CheckCommandAuthorization(guildID, userID, targetID, commandName string) (bool, string) {
	// Bot owners always authorized
	if sf.permChecker.IsBotOwner(userID) {
		return false, ""
	}

	// Check if auto-ban on unauthorized commands is enabled
	if !Global.SpamFilter.AutoBanOnUnauthorizedCommands {
		return false, ""
	}

	// Check if user has required permission
	var requiredPerm int64 = discordgo.PermissionBanMembers
	if commandName == "kick" {
		requiredPerm = discordgo.PermissionKickMembers
	}

	if !sf.permChecker.HasPermission(guildID, userID, requiredPerm) {
		DebugLog("User %s tried to use %s without permission", userID, commandName)
		return true, "Attempting to use moderation commands without permission"
	}

	// If target provided, check hierarchy violation
	if targetID != "" && Global.SpamFilter.AutoBanOnHierarchyViolation {
		canModerate, reason := sf.permChecker.CanModerate(guildID, userID, targetID)
		if !canModerate {
			DebugLog("User %s hierarchy violation: %s", userID, reason)
			return true, "Attempting to moderate higher-ranked user: " + reason
		}
	}

	return false, ""
}
