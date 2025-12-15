package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// BanEntry represents a ban record for export/import
type BanEntry struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Reason   string    `json:"reason"`
	BannedAt time.Time `json:"banned_at,omitempty"`
}

// ExportBansCommand exports the server's ban list
type ExportBansCommand struct{}

func (c *ExportBansCommand) Name() string        { return "exportbans" }
func (c *ExportBansCommand) Aliases() []string   { return []string{"export-bans", "bansexport"} }
func (c *ExportBansCommand) Description() string { return "Export the server's ban list to JSON" }
func (c *ExportBansCommand) Usage() string       { return "exportbans" }
func (c *ExportBansCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *ExportBansCommand) MasterOnly() bool { return false }

func (c *ExportBansCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Fetching ban list...")

	// Get all bans
	bans, err := ctx.Session.GuildBans(ctx.Message.GuildID, 0, "", "")
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error fetching ban list.")
		return err
	}

	if len(bans) == 0 {
		if msg != nil {
			ctx.Session.ChannelMessageEdit(ctx.Message.ChannelID, msg.ID, "No bans found in this server.")
		}
		return nil
	}

	// Convert to export format
	entries := make([]BanEntry, 0, len(bans))
	for _, ban := range bans {
		entries = append(entries, BanEntry{
			UserID:   ban.User.ID,
			Username: ban.User.Username,
			Reason:   ban.Reason,
			BannedAt: time.Now(), // Discord doesn't provide ban timestamp
		})
	}

	// Create JSON
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	// Send as file
	guild, _ := ctx.Session.Guild(ctx.Message.GuildID)
	guildName := "server"
	if guild != nil {
		guildName = guild.Name
	}

	filename := fmt.Sprintf("%s_bans_%s.json",
		strings.ReplaceAll(guildName, " ", "_"),
		time.Now().Format("2006-01-02"))

	// Delete the loading message
	if msg != nil {
		ctx.Session.ChannelMessageDelete(ctx.Message.ChannelID, msg.ID)
	}

	// Send file
	_, err = ctx.Session.ChannelFileSend(ctx.Message.ChannelID, filename, strings.NewReader(string(jsonData)))
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error sending ban list file.")
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Bans Exported",
		Description: fmt.Sprintf("Exported %d bans to `%s`", len(entries), filename),
		Color:       0x00FF00,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ImportBansCommand imports bans from a JSON file
type ImportBansCommand struct{}

func (c *ImportBansCommand) Name() string        { return "importbans" }
func (c *ImportBansCommand) Aliases() []string   { return []string{"import-bans", "bansimport"} }
func (c *ImportBansCommand) Description() string { return "Import bans from a JSON file" }
func (c *ImportBansCommand) Usage() string       { return "importbans (with attached JSON file)" }
func (c *ImportBansCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *ImportBansCommand) MasterOnly() bool { return false }

func (c *ImportBansCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Message.Attachments) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Please attach a JSON file containing the ban list.\n"+
				"Format: `[{\"user_id\": \"123\", \"reason\": \"...\"}]`")
		return nil
	}

	attachment := ctx.Message.Attachments[0]
	if !strings.HasSuffix(strings.ToLower(attachment.Filename), ".json") {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Please attach a JSON file.")
		return nil
	}

	// Download the file
	resp, err := httpClient.Get(attachment.URL)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error downloading file.")
		return err
	}
	defer resp.Body.Close()

	var entries []BanEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"Error parsing JSON file. Ensure it's in the correct format.")
		return err
	}

	if len(entries) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "No ban entries found in the file.")
		return nil
	}

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
		fmt.Sprintf("Importing %d bans...", len(entries)))

	imported := 0
	skipped := 0
	errors := 0

	for _, entry := range entries {
		if entry.UserID == "" {
			skipped++
			continue
		}

		reason := entry.Reason
		if reason == "" {
			reason = "Imported ban"
		}
		reason = fmt.Sprintf("[Import] %s | Imported by %s", reason, ctx.Message.Author.Username)

		err := ctx.Session.GuildBanCreateWithReason(ctx.Message.GuildID, entry.UserID, reason, 0)
		if err != nil {
			// Check if already banned
			if strings.Contains(err.Error(), "already banned") {
				skipped++
			} else {
				errors++
			}
			continue
		}
		imported++
	}

	embed := &discordgo.MessageEmbed{
		Title: "✅ Bans Imported",
		Color: 0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Imported", Value: strconv.Itoa(imported), Inline: true},
			{Name: "Skipped", Value: strconv.Itoa(skipped), Inline: true},
			{Name: "Errors", Value: strconv.Itoa(errors), Inline: true},
		},
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}

// ScanBansCommand scans and validates the ban list
type ScanBansCommand struct{}

func (c *ScanBansCommand) Name() string        { return "scan-bans" }
func (c *ScanBansCommand) Aliases() []string   { return []string{"scanbans", "checkbans"} }
func (c *ScanBansCommand) Description() string { return "Scan and show statistics about the ban list" }
func (c *ScanBansCommand) Usage() string       { return "scan-bans" }
func (c *ScanBansCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *ScanBansCommand) MasterOnly() bool { return false }

func (c *ScanBansCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	msg, _ := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Scanning ban list...")

	bans, err := ctx.Session.GuildBans(ctx.Message.GuildID, 0, "", "")
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error fetching ban list.")
		return err
	}

	if len(bans) == 0 {
		if msg != nil {
			ctx.Session.ChannelMessageEdit(ctx.Message.ChannelID, msg.ID, "No bans found in this server.")
		}
		return nil
	}

	// Analyze bans
	withReason := 0
	withoutReason := 0
	deletedUsers := 0

	for _, ban := range bans {
		if ban.Reason != "" {
			withReason++
		} else {
			withoutReason++
		}
		// Check if user account is deleted (discriminator = 0000 in old API)
		// In newer API, we check if username is "Deleted User"
		if strings.HasPrefix(ban.User.Username, "Deleted User") {
			deletedUsers++
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Ban List Analysis",
		Color: 0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Total Bans", Value: strconv.Itoa(len(bans)), Inline: true},
			{Name: "With Reason", Value: strconv.Itoa(withReason), Inline: true},
			{Name: "Without Reason", Value: strconv.Itoa(withoutReason), Inline: true},
			{Name: "Deleted Accounts", Value: strconv.Itoa(deletedUsers), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Use %sexportbans to export the full list", ctx.GetPrefix()),
		},
	}

	if msg != nil {
		ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, msg.ID, embed)
	} else {
		ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	}

	return nil
}
