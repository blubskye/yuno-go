package bot

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"yuno-bot/internal/commands"
	"yuno-bot/internal/config"
	"yuno-bot/internal/database"
)

type Bot struct {
	Config      *config.Config
	DB          *database.Database
	Commands    *commands.Manager
	Session     *discordgo.Session
	terminalQuit chan bool
}

// New creates a new bot instance
func New(cfg *config.Config, db *database.Database) (*Bot, error) {
	bot := &Bot{
		Config:       cfg,
		DB:           db,
		terminalQuit: make(chan bool),
	}

	// Initialize command manager
	bot.Commands = commands.NewManager(cfg, db)
	
	// Register commands
	bot.Commands.Register(&commands.PingCommand{})
	bot.Commands.Register(&commands.XPCommand{})
	bot.Commands.Register(&commands.SetLevelCommand{})
	bot.Commands.Register(&commands.BanCommand{})
	bot.Commands.Register(&commands.KickCommand{})
	bot.Commands.Register(&commands.StatsCommand{})
	bot.Commands.Register(&commands.ShutdownCommand{})

	return bot, nil
}

// RegisterHandlers registers all Discord event handlers
func (b *Bot) RegisterHandlers(s *discordgo.Session) {
	b.Session = s
	
	s.AddHandler(b.onReady)
	s.AddHandler(b.onMessageCreate)
	s.AddHandler(b.onGuildMemberAdd)
}

// onReady handles the ready event
func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as %s#%s", event.User.Username, event.User.Discriminator)
	log.Printf("Bot is in %d guilds", len(event.Guilds))

	// Set presence if configured
	if b.Config.Discord.Presence != nil {
		activities := make([]*discordgo.Activity, len(b.Config.Discord.Presence.Activities))
		for i, act := range b.Config.Discord.Presence.Activities {
			activities[i] = &discordgo.Activity{
				Name: act.Name,
				Type: discordgo.ActivityType(act.Type),
				URL:  act.URL,
			}
		}

		err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: activities,
			Status:     b.Config.Discord.Presence.Status,
		})
		if err != nil {
			log.Printf("Failed to set presence: %v", err)
		}
	}
}

// onMessageCreate handles message creation
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	// Handle DMs differently
	if m.GuildID == "" {
		b.handleDM(s, m)
		return
	}

	// Get guild prefix
	prefix, err := b.DB.GetPrefix(m.GuildID)
	if err != nil {
		log.Printf("Error getting prefix: %v", err)
		return
	}

	// Use default prefix if none set
	if prefix == "" {
		prefix = b.Config.Commands.DefaultPrefix
	}

	// Check if message starts with prefix
	if !strings.HasPrefix(m.Content, prefix) {
		// Handle XP gain if enabled
		b.handleXPGain(s, m)
		return
	}

	// Parse and execute command
	content := strings.TrimPrefix(m.Content, prefix)
	ctx := &commands.Context{
		Session: s,
		Message: m,
		Bot:     b,
	}

	if err := b.Commands.Execute(ctx, content); err != nil {
		log.Printf("Command error: %v", err)
	}
}

// handleDM handles direct messages
func (b *Bot) handleDM(s *discordgo.Session, m *discordgo.MessageCreate) {
	dmMsg := b.Config.Chat.DM
	if dmMsg == "" {
		dmMsg = "I'm just a bot :'(. I can't answer to you."
	}

	s.ChannelMessageSend(m.ChannelID, dmMsg)
}

// handleXPGain processes XP gain for messages
func (b *Bot) handleXPGain(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if XP is enabled for this guild
	enabled, err := b.DB.IsXPEnabled(m.GuildID)
	if err != nil || !enabled {
		return
	}

	// Get current XP
	xpData, err := b.DB.GetXPData(m.GuildID, m.Author.ID)
	if err != nil {
		return
	}

	// Add random XP (15-25)
	newXP := xpData.XP + 15 + (m.ID[0] % 11) // Pseudo-random based on message ID

	// Calculate level
	level := xpData.Level
	neededXP := 5*level*level + 50*level + 100

	if newXP >= neededXP {
		level++
		newXP -= neededXP
		
		// Send level up message
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🎉 <@%s> leveled up to level **%d**!", m.Author.ID, level))
	}

	// Update database
	b.DB.SetXPData(m.GuildID, m.Author.ID, newXP, level)
}

// onGuildMemberAdd handles new member joins
func (b *Bot) onGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	log.Printf("New member joined: %s in guild %s", m.User.Username, m.GuildID)
	// TODO: Implement welcome DM if configured
}

// StartInteractiveTerminal starts the interactive terminal
func (b *Bot) StartInteractiveTerminal() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		
		if input == "" {
			fmt.Print("> ")
			continue
		}

		// Handle special commands
		switch input {
		case "quit", "exit":
			fmt.Println("Shutting down...")
			b.terminalQuit <- true
			return
		case "help":
			b.showTerminalHelp()
		default:
			// Execute as bot command
			ctx := &commands.Context{
				Session: b.Session,
				Bot:     b,
			}
			if err := b.Commands.Execute(ctx, input); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}

		fmt.Print("> ")
	}
}

func (b *Bot) showTerminalHelp() {
	fmt.Println("Available terminal commands:")
	fmt.Println("  help       - Show this help")
	fmt.Println("  quit/exit  - Shutdown the bot")
	fmt.Println("  stats      - Show bot statistics")
	fmt.Println("  shutdown   - Graceful shutdown")
	fmt.Println("\nYou can also run any bot command here without a prefix.")
}

// GetDB returns the database instance
func (b *Bot) GetDB() *database.Database {
	return b.DB
}

// Shutdown performs graceful shutdown
func (b *Bot) Shutdown() {
	log.Println("Saving configuration...")
	// Add any cleanup needed
	log.Println("Shutdown complete")
}