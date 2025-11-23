package bot

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
)

type Bot struct {
	Session *discordgo.Session
	DB      *Database
}

func New() (*Bot, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	db, err := NewDatabase("Leveling/main.db")
	if err != nil {
		return nil, err
	}

	b := &Bot{
		Session: dg,
		DB:      db,
	}

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

func (b *Bot) Start() error {
	return b.Session.Open()
}

func (b *Bot) Stop() {
	b.Session.Close()
	b.DB.Close()
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("→ Logged in as %s | Serving %d guilds", s.State.User.String(), len(r.Guilds))
	s.UpdateWatchingStatus(0, "for levels ♡")
}
