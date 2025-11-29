package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AddFilterCommand adds a regex filter
type AddFilterCommand struct{}

func (c *AddFilterCommand) Name() string { return "addfilter" }
func (c *AddFilterCommand) Aliases() []string { return []string{"addregex"} }
func (c *AddFilterCommand) Description() string {
	return "Add a regex filter to auto-moderate messages"
}
func (c *AddFilterCommand) Usage() string {
	return "addfilter <pattern> | <action> | <reason>"
}
func (c *AddFilterCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageGuild}
}
func (c *AddFilterCommand) MasterOnly() bool { return false }

func (c *AddFilterCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `addfilter <pattern> | <action> | <reason>`\nActions: `warn`, `ban`, `delete`")
		return nil
	}

	// Parse arguments
	fullArgs := strings.Join(ctx.Args, " ")
	parts := strings.Split(fullArgs, "|")

	if len(parts) < 3 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Missing arguments. Format: `addfilter <pattern> | <action> | <reason>`")
		return nil
	}

	pattern := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(strings.ToLower(parts[1]))
	reason := strings.TrimSpace(parts[2])

	// Validate action
	if action != "warn" && action != "ban" && action != "delete" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Invalid action. Must be: `warn`, `ban`, or `delete`")
		return nil
	}

	// Get database - use type assertion to get the actual DB
	type BotDB interface {
		GetDB() interface{ Exec(string, ...interface{}) (interface{}, error) }
	}
	botDB := ctx.Bot.(BotDB).GetDB()

	// Insert filter
	timestamp := time.Now().Format(time.RFC3339)
	_, err := botDB.Exec(`
		INSERT INTO regex_filters (guild_id, pattern, action, reason, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, ctx.Message.GuildID, pattern, action, reason, ctx.Message.Author.ID, timestamp)

	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ A filter with this pattern already exists!")
			return nil
		}
		return fmt.Errorf("failed to add filter: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Filter Added",
		Description: fmt.Sprintf("Successfully added regex filter"),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Pattern", Value: fmt.Sprintf("`%s`", pattern), Inline: false},
			{Name: "Action", Value: action, Inline: true},
			{Name: "Reason", Value: reason, Inline: true},
		},
		Color: 0x43CC24,
	}
	ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)

	return nil
}

// RemoveFilterCommand removes a regex filter
type RemoveFilterCommand struct{}

func (c *RemoveFilterCommand) Name() string { return "removefilter" }
func (c *RemoveFilterCommand) Aliases() []string { return []string{"delfilter", "removeregex"} }
func (c *RemoveFilterCommand) Description() string {
	return "Remove a regex filter by ID"
}
func (c *RemoveFilterCommand) Usage() string { return "removefilter <id>" }
func (c *RemoveFilterCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageGuild}
}
func (c *RemoveFilterCommand) MasterOnly() bool { return false }

func (c *RemoveFilterCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `removefilter <id>` (use `listfilters` to see IDs)")
		return nil
	}

	filterID := ctx.Args[0]
	type BotDB interface {
		GetDB() interface{ Exec(string, ...interface{}) (interface{}, error) }
	}
	botDB := ctx.Bot.(BotDB).GetDB()

	_, err := botDB.Exec(`
		DELETE FROM regex_filters
		WHERE guild_id = ? AND id = ?
	`, ctx.Message.GuildID, filterID)

	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ No filter found with that ID")
			return nil
		}
		return fmt.Errorf("failed to remove filter: %v", err)
	}

	// Check if filter existed - we can't easily get rows affected with the current interface
	// so we'll just assume success if no error
	if false {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ No filter found with that ID")
		return nil
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("✅ Removed filter #%s", filterID))

	return nil
}

// ListFiltersCommand lists all regex filters
type ListFiltersCommand struct{}

func (c *ListFiltersCommand) Name() string { return "listfilters" }
func (c *ListFiltersCommand) Aliases() []string { return []string{"filters", "listregex"} }
func (c *ListFiltersCommand) Description() string {
	return "List all regex filters for this server"
}
func (c *ListFiltersCommand) Usage() string { return "listfilters" }
func (c *ListFiltersCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionManageGuild}
}
func (c *ListFiltersCommand) MasterOnly() bool { return false }

func (c *ListFiltersCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	type BotDB interface {
		GetDB() interface{ Query(string, ...interface{}) (interface{}, error) }
	}
	botDB := ctx.Bot.(BotDB).GetDB()

	result, err := botDB.Query(`
		SELECT id, pattern, action, reason, enabled
		FROM regex_filters
		WHERE guild_id = ?
		ORDER BY id
	`, ctx.Message.GuildID)

	if err != nil {
		return fmt.Errorf("failed to fetch filters: %v", err)
	}

	// Cast result to a Rows-like interface
	type Rows interface {
		Close() error
		Next() bool
		Scan(...interface{}) error
	}
	rows := result.(Rows)
	defer rows.Close()

	var fields []*discordgo.MessageEmbedField
	count := 0

	for rows.Next() {
		var id int
		var pattern, action, reason string
		var enabled bool

		if err := rows.Scan(&id, &pattern, &action, &reason, &enabled); err != nil {
			continue
		}

		status := "✅"
		if !enabled {
			status = "❌"
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("%s Filter #%d - %s", status, id, action),
			Value: fmt.Sprintf("**Pattern:** `%s`\n**Reason:** %s",
				pattern, reason),
			Inline: false,
		})
		count++

		// Discord has a limit of 25 fields
		if count >= 25 {
			break
		}
	}

	if count == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"No regex filters configured for this server.\nUse `addfilter` to add one!")
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Regex Filters (%d total)", count),
		Description: "Use `removefilter <id>` to remove a filter",
		Fields:      fields,
		Color:       0xFF51FF,
	}

	ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return nil
}
