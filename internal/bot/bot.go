package bot

import (
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
	"yuno-go/internal/commands"
)

type Bot struct {
	Session         *discordgo.Session
	DB              *Database
	Commands        *commands.Manager
	CleanWorker     *AutoCleanWorker
	PresenceBatcher *PresenceBatcher
	SpamFilter      *SpamFilter
	PermChecker     *PermissionChecker
}

// RecoverFromPanic recovers from panics and logs stack trace if enabled
func RecoverFromPanic(context string) {
	if r := recover(); r != nil {
		log.Printf("⚠️  PANIC in %s: %v", context, r)

		if Global.Debug.PrintStackOnPanic || Global.Debug.FullStackTrace {
			log.Printf("Stack trace:\n%s", debug.Stack())
		}
	}
}

func New() (*Bot, error) {
	defer RecoverFromPanic("Bot.New")

	DebugLog("Initializing bot...")

	// Get token from config (already loaded by main.go)
	token := Global.Bot.Token
	if token == "" {
		log.Fatal("Bot token not set in config")
	}

	DebugLog("Creating Discord session...")
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	// Use database path from config
	dbPath := Global.Database.Path
	if dbPath == "" {
		dbPath = "Leveling/main.db"
	}

	DebugLog("Opening database at: %s", dbPath)
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	DebugLog("Database initialized successfully")

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

	// Initialize presence batcher
	b.PresenceBatcher = NewPresenceBatcher(b)

	// Initialize spam filter
	b.SpamFilter = NewSpamFilter(b)
	DebugLog("Spam filter initialized")

	// Initialize permission checker
	b.PermChecker = NewPermissionChecker(dg)
	DebugLog("Permission checker initialized")

	// Register all commands
	DebugLog("Registering commands...")
	b.registerCommands()

	DebugLog("Adding event handlers...")
	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onMessageCreate)
	dg.AddHandler(b.onVoiceStateUpdate)
	dg.AddHandler(b.onMemberJoin)

	// Logging handlers
	dg.AddHandler(b.onMessageDelete)
	dg.AddHandler(b.onMessageUpdate)
	dg.AddHandler(b.onVoiceStateUpdateLogging)
	dg.AddHandler(b.onGuildMemberUpdate)
	dg.AddHandler(b.onPresenceUpdate)

	// All intents we need
	dg.Identify.Intents = discordgo.IntentsAllWithoutPrivileged |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildPresences

	DebugLog("Bot initialization complete")
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
	b.Commands.Register(&commands.SyncLevelsCommand{})
	b.Commands.Register(&commands.PreviewLevelsCommand{})
	
	// Rank role management
	b.Commands.Register(&commands.SyncRanksCommand{})
	b.Commands.Register(&commands.AddRankCommand{})
	b.Commands.Register(&commands.RemoveRankCommand{})
	b.Commands.Register(&commands.ListRanksCommand{})
	b.Commands.Register(&commands.ApplyRanksCommand{})
	
	// Moderation commands
	b.Commands.Register(&commands.BanCommand{})
	b.Commands.Register(&commands.KickCommand{})

	// Spam filter commands
	b.Commands.Register(&commands.AddFilterCommand{})
	b.Commands.Register(&commands.RemoveFilterCommand{})
	b.Commands.Register(&commands.ListFiltersCommand{})

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

	// Logging commands
	b.Commands.Register(&commands.SetLogChannelCommand{})
	b.Commands.Register(&commands.ToggleLoggingCommand{})
	b.Commands.Register(&commands.ConfigureLogTypeCommand{})
	b.Commands.Register(&commands.SetPresenceBatchCommand{})
	b.Commands.Register(&commands.DisableChannelLoggingCommand{})
	b.Commands.Register(&commands.EnableChannelLoggingCommand{})
	b.Commands.Register(&commands.LogStatusCommand{})

	log.Printf("Registered %d commands", len(b.Commands.GetAll()))
}

func (b *Bot) Start() error {
	// Start auto-clean worker
	b.CleanWorker.Start()

	// Start presence batcher
	b.PresenceBatcher.Start()

	return b.Session.Open()
}

func (b *Bot) Stop() {
	b.CleanWorker.Stop()
	b.PresenceBatcher.Stop()
	b.Session.Close()
	b.DB.Close()
}

// GetDB returns the underlying sql.DB for commands to use
func (b *Bot) GetDB() *Database {
	return b.DB
}

func (b *Bot) GetSpamFilter() *SpamFilter {
	return b.SpamFilter
}

func (b *Bot) GetPermChecker() *PermissionChecker {
	return b.PermChecker
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	defer RecoverFromPanic("onReady")

	log.Printf("→ Logged in as %s | Serving %d guilds", s.State.User.String(), len(r.Guilds))
	DebugLog("Ready event received with %d guilds", len(r.Guilds))

	if Global.Debug.PrintRawEvents {
		log.Printf("[RAW EVENT] Ready: User=%s, SessionID=%s", r.User.String(), r.SessionID)
	}

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

	// Start daily cleanup task for message cache
	go b.dailyCleanupTask()
}

func (b *Bot) dailyCleanupTask() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once immediately
	b.CleanOldMessageCache()

	for range ticker.C {
		b.CleanOldMessageCache()
	}
}

// onMessageCreate now handles commands
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer RecoverFromPanic("onMessageCreate")

	if m.Author.Bot || m.GuildID == "" {
		return
	}

	if Global.Debug.PrintRawEvents {
		log.Printf("[RAW EVENT] Message: Author=%s, Guild=%s, Content=%s",
			m.Author.Username, m.GuildID, m.Content)
	}

	// Check if message starts with prefix
	prefix := Global.Bot.Prefix
	if prefix == "" {
		prefix = "?"
	}

	// Check spam filter first (before commands)
	if filterResult := b.SpamFilter.CheckMessage(m); filterResult != nil {
		DebugLog("Message triggered spam filter: %s", filterResult.Reason)
		b.SpamFilter.ExecuteAction(s, m, filterResult)
		return
	}

	if strings.HasPrefix(m.Content, prefix) {
		// Handle command
		content := strings.TrimPrefix(m.Content, prefix)
		DebugLog("Command received: %s from %s", content, m.Author.Username)

		// Check for moderation command authorization
		cmdParts := strings.Fields(content)
		if len(cmdParts) > 0 {
			cmdName := strings.ToLower(cmdParts[0])
			if cmdName == "ban" || cmdName == "kick" {
				// Get first target ID
				var targetID string
				if len(m.Mentions) > 0 {
					targetID = m.Mentions[0].ID
				}

				// Check authorization
				shouldBan, reason := b.SpamFilter.CheckCommandAuthorization(
					m.GuildID,
					m.Author.ID,
					targetID,
					cmdName,
				)
				if shouldBan {
					DebugLog("Auto-banning %s for unauthorized command: %s", m.Author.ID, reason)
					b.PermChecker.AutoBanViolator(m.GuildID, m.Author.ID, reason)
					return
				}
			}
		}

		ctx := &commands.Context{
			Session: s,
			Message: m,
			Bot:     b,
		}

		if err := b.Commands.Execute(ctx, content); err != nil {
			log.Printf("Command error: %v", err)
			if Global.Debug.VerboseLogging {
				log.Printf("Command execution failed for '%s': %v", content, err)
			}
		}
		return
	}

	// Cache message for logging
	b.CacheMessage(m.Message)

	// Give XP if not a command
	b.giveXPAsync(s, m)
}

func (b *Bot) giveXPAsync(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Import rand at top if not already imported
	xp := 15 + (int(m.ID[0]) % 11) // Pseudo-random 15-25 XP based on message ID
	go b.giveXP(s, m.GuildID, m.Author.ID, m.ChannelID, xp)
}
