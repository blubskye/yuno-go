package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
	"yuno-go/internal/commands"
)

type Bot struct {
	Session       *discordgo.Session
	DB            *Database
	Commands      *commands.Manager
	CleanWorker   *AutoCleanWorker
}

func New() (*Bot, error) {
	// Get token from config (already loaded by main.go)
	token := Global.Bot.Token
	if token == "" {
		log.Fatal("Bot token not set in config")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	// Use database path from config
	dbPath := Global.Database.Path
	if dbPath == "" {
		dbPath = "Leveling/main.db"
	}

	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		Session:  dg,
		DB:       db,
		Commands: commands.NewManager(
			Global.Bot.Prefix,
			Global.Bot.OwnerIDs,
			Global.AGPL.SourceURL,
			Global.Paths.BanImagesFolder,
		),
	}

	// Initialize auto-clean worker
	b.CleanWorker = NewAutoCleanWorker(b)

	// Register all commands
	b.registerCommands()

	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onMessageCreate)
	dg.AddHandler(b.onVoiceStateUpdate)
	dg.AddHandler(b.onMemberJoin)

	// All intents we need
	dg.Identify.Intents = discordgo.IntentsAllWithoutPrivileged |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsMessageContent

	return b, nil
}

func (b *Bot) registerCommands() {
	// Basic commands
	b.Commands.Register(&commands.PingCommand{})
	b.Commands.Register(&commands.StatsCommand{})
	b.Commands.Register(&commands.HelpCommand{})
	
	// Leveling commands
	b.Commands.Register(&commands.XPCommand{})
	b.Commands.Register(&commands.SetLevelCommand{})
	
	// Moderation commands
	b.Commands.Register(&commands.BanCommand{})
	b.Commands.Register(&commands.KickCommand{})
	
	// Owner commands
	b.Commands.Register(&commands.ShutdownCommand{})
	
	// Auto-clean commands
	b.Commands.Register(&commands.AutoCleanCommand{})
	b.Commands.Register(&commands.SetCleanMessageCommand{})
	b.Commands.Register(&commands.SetCleanImageCommand{})
	
	// Source & Ban image commands
	b.Commands.Register(&commands.SourceCommand{})
	b.Commands.Register(&commands.AddBanImageCommand{})
	b.Commands.Register(&commands.DelBanImageCommand{})

	log.Printf("Registered %d commands", len(b.Commands.GetAll()))
}

func (b *Bot) Start() error {
	// Start auto-clean worker
	b.CleanWorker.Start()
	return b.Session.Open()
}

func (b *Bot) Stop() {
	b.CleanWorker.Stop()
	b.Session.Close()
	b.DB.Close()
}

// GetDB returns the underlying sql.DB for commands to use
func (b *Bot) GetDB() *Database {
	return b.DB
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("→ Logged in as %s | Serving %d guilds", s.State.User.String(), len(r.Guilds))
	
	// Use UpdateStatusComplex instead of deprecated UpdateWatchingStatus
	activityType := discordgo.ActivityTypeWatching
	status := Global.Bot.Status
	if status == "" {
		status = "for levels ♡"
	}

	err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{{
			Name: status,
			Type: activityType,
		}},
		Status: "online",
	})
	
	if err != nil {
		log.Printf("Warning: Could not set status: %v", err)
	}
}

// onMessageCreate now handles commands
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.GuildID == "" {
		return
	}

	// Check if message starts with prefix
	prefix := Global.Bot.Prefix
	if prefix == "" {
		prefix = "?"
	}

	if strings.HasPrefix(m.Content, prefix) {
		// Handle command
		content := strings.TrimPrefix(m.Content, prefix)
		ctx := &commands.Context{
			Session: s,
			Message: m,
			Bot:     b,
		}

		if err := b.Commands.Execute(ctx, content); err != nil {
			log.Printf("Command error: %v", err)
		}
		return
	}

	// Give XP if not a command
	b.giveXPAsync(s, m)
}

func (b *Bot) giveXPAsync(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Import rand at top if not already imported
	xp := 15 + (int(m.ID[0]) % 11) // Pseudo-random 15-25 XP based on message ID
	go b.giveXP(s, m.GuildID, m.Author.ID, m.ChannelID, xp)
}
