package commands

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SourceCommand provides source code link (AGPL compliance)
type SourceCommand struct{}

func (c *SourceCommand) Name() string        { return "source" }
func (c *SourceCommand) Aliases() []string   { return []string{"src", "code", "github"} }
func (c *SourceCommand) Description() string { return "Get the bot's source code (AGPL v3)" }
func (c *SourceCommand) Usage() string       { return "source" }
func (c *SourceCommand) RequiredPermissions() []int64 { return nil }
func (c *SourceCommand) MasterOnly() bool    { return false }

func (c *SourceCommand) Execute(ctx *Context) error {
	// Get source URL from context helper
	sourceURL := ctx.GetSourceURL()
	if sourceURL == "" {
		sourceURL = "https://github.com/blubskye/yuno-go"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Yuno Gasai - Source Code",
		Description: "This bot is open source and licensed under **GNU AGPL v3**.",
		Color:       0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "📦 Repository",
				Value: fmt.Sprintf("[View on GitHub](%s)", sourceURL),
			},
			{
				Name:  "📜 License",
				Value: "GNU Affero General Public License v3.0",
			},
			{
				Name:  "ℹ️ What does this mean?",
				Value: "You have the freedom to use, study, modify, and distribute this software. If you run a modified version, you must also make your source code available.",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Made with ♡ in Go",
		},
	}

	// Try to DM first
	if ctx.Message != nil {
		dmChannel, err := ctx.Session.UserChannelCreate(ctx.Message.Author.ID)
		if err == nil {
			_, dmErr := ctx.Session.ChannelMessageSendEmbed(dmChannel.ID, embed)
			if dmErr == nil {
				// Successfully sent DM
				if ctx.Message.GuildID != "" {
					// Confirm in channel
					ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
						fmt.Sprintf("📬 %s Check your DMs for the source code link!", ctx.Message.Author.Mention()))
					return nil
				}
				return nil
			}
		}

		// DM failed, send in channel
		_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Terminal execution
	fmt.Printf("Source Code: %s\n", sourceURL)
	fmt.Println("License: GNU AGPL v3")
	return nil
}

// AddBanImageCommand adds custom ban images
type AddBanImageCommand struct{}

func (c *AddBanImageCommand) Name() string      { return "add-ban-image" }
func (c *AddBanImageCommand) Aliases() []string { return []string{"addbanimg", "abi"} }
func (c *AddBanImageCommand) Description() string {
	return "Add a custom ban image (from URL or attachment)"
}
func (c *AddBanImageCommand) Usage() string {
	return "add-ban-image <url|attachment>"
}
func (c *AddBanImageCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *AddBanImageCommand) MasterOnly() bool { return false }

func (c *AddBanImageCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	var imageURL string

	// Check for attachment first
	if len(ctx.Message.Attachments) > 0 {
		attachment := ctx.Message.Attachments[0]
		// Check if it's an image
		if !strings.HasPrefix(attachment.ContentType, "image/") {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ Please attach an image file (PNG, JPG, GIF)")
			return nil
		}
		imageURL = attachment.URL
	} else if len(ctx.Args) > 0 {
		// Check for URL argument
		imageURL = ctx.Args[0]
		if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
				"❌ Please provide a valid image URL or attach an image")
			return nil
		}
	} else {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Usage: `add-ban-image <url>` or attach an image")
		return nil
	}

	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to download image: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to download image: HTTP %d", resp.StatusCode))
		return nil
	}

	// Create ban images directory
	banImagesPath := ctx.GetBanImagesPath()
	if banImagesPath == "" {
		banImagesPath = "assets/ban_images"
	}
	
	guildBanPath := filepath.Join(banImagesPath, ctx.Message.GuildID)
	err = os.MkdirAll(guildBanPath, 0755)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to create directory: %v", err))
		return err
	}

	// Generate unique filename
	ext := filepath.Ext(imageURL)
	if ext == "" || len(ext) > 5 {
		ext = ".png"
	}
	filename := fmt.Sprintf("ban_%d%s", ctx.Message.ID, ext)
	filepath := filepath.Join(guildBanPath, filename)

	// Save the image
	out, err := os.Create(filepath)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to save image: %v", err))
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to write image: %v", err))
		return err
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Add to database
	_, err = db.Exec(`
		INSERT INTO ban_images (guild_id, filename, added_by, added_at)
		VALUES (?, ?, ?, datetime('now'))`,
		ctx.Message.GuildID, filename, ctx.Message.Author.ID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to save to database: %v", err))
		return err
	}

	// Success message with preview
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Ban Image Added",
		Description: fmt.Sprintf("Added by %s", ctx.Message.Author.Mention()),
		Color:       0x43CC24,
		Image: &discordgo.MessageEmbedImage{
			URL: imageURL,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Filename: %s", filename),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// DelBanImageCommand removes custom ban images
type DelBanImageCommand struct{}

func (c *DelBanImageCommand) Name() string      { return "del-ban-image" }
func (c *DelBanImageCommand) Aliases() []string { return []string{"delbanimg", "dbi", "removebanimg"} }
func (c *DelBanImageCommand) Description() string {
	return "Remove a custom ban image by number"
}
func (c *DelBanImageCommand) Usage() string {
	return "del-ban-image <number>"
}
func (c *DelBanImageCommand) RequiredPermissions() []int64 {
	return []int64{discordgo.PermissionBanMembers}
}
func (c *DelBanImageCommand) MasterOnly() bool { return false }

func (c *DelBanImageCommand) Execute(ctx *Context) error {
	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return fmt.Errorf("this command can only be used in a server")
	}

	if len(ctx.Args) == 0 {
		return c.listBanImages(ctx)
	}

	// Parse image number
	var imageNum int
	_, err := fmt.Sscanf(ctx.Args[0], "%d", &imageNum)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Please provide a valid image number. Use `del-ban-image` to see the list.")
		return nil
	}

	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	// Get the filename for this number
	var filename string
	var id int
	err = db.QueryRow(`
		SELECT id, filename FROM ban_images 
		WHERE guild_id = ? 
		ORDER BY added_at ASC 
		LIMIT 1 OFFSET ?`,
		ctx.Message.GuildID, imageNum-1).Scan(&id, &filename)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			"❌ Ban image not found. Use `del-ban-image` to see the list.")
		return nil
	}

	// Delete from database
	_, err = db.Exec(`DELETE FROM ban_images WHERE id = ?`, id)
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to delete from database: %v", err))
		return err
	}

	// Delete file
	banImagesPath := ctx.GetBanImagesPath()
	if banImagesPath == "" {
		banImagesPath = "assets/ban_images"
	}
	filePath := filepath.Join(banImagesPath, ctx.Message.GuildID, filename)
	os.Remove(filePath) // Ignore errors if file doesn't exist

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Ban Image Removed",
		Description: fmt.Sprintf("Deleted ban image #%d", imageNum),
		Color:       0x43CC24,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *DelBanImageCommand) listBanImages(ctx *Context) error {
	// Get database
	db := ctx.Bot.(interface{ GetDB() DBInterface }).GetDB()

	rows, err := db.(interface {
		Query(query string, args ...interface{}) (interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
		}, error)
	}).Query(`
		SELECT filename, added_by, added_at 
		FROM ban_images 
		WHERE guild_id = ? 
		ORDER BY added_at ASC`,
		ctx.Message.GuildID)

	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID,
			fmt.Sprintf("❌ Failed to list ban images: %v", err))
		return err
	}
	defer rows.Close()

	embed := &discordgo.MessageEmbed{
		Title:       "Ban Images List",
		Description: "Use `del-ban-image <number>` to remove an image",
		Color:       0xFF51FF,
		Fields:      []*discordgo.MessageEmbedField{},
	}

	count := 0
	for rows.Next() {
		var filename, addedBy, addedAt string
		if err := rows.Scan(&filename, &addedBy, &addedAt); err != nil {
			continue
		}

		count++
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("#%d - %s", count, filename),
			Value:  fmt.Sprintf("Added by <@%s>", addedBy),
			Inline: false,
		})
	}

	if count == 0 {
		embed.Description = "No custom ban images. Add one with `add-ban-image <url>` or attach an image."
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// GetRandomBanImage returns a random ban image path for a guild
func GetRandomBanImage(guildID string, banImagesPath string) string {
	if banImagesPath == "" {
		banImagesPath = "assets/ban_images"
	}

	guildPath := filepath.Join(banImagesPath, guildID)
	files, err := os.ReadDir(guildPath)
	if err != nil || len(files) == 0 {
		return "" // No custom images
	}

	// Get random file
	randomFile := files[rand.Intn(len(files))]
	return filepath.Join(guildPath, randomFile.Name())
}
