package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"yuno-go/internal/bot"
)

// Global config will be loaded here
func main() {
	// 1. Load configuration (creates default config.toml if missing)
	bot.LoadConfig("config.toml")

	// 2. Create the bot instance
	yuno, err := bot.New()
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// 3. Set custom status from config
	activityType := discordgo.ActivityTypeWatching
	switch bot.Global.Bot.ActivityType {
	case "playing":
		activityType = discordgo.ActivityTypeGame
	case "streaming":
		activityType = discordgo.ActivityTypeStreaming
	case "listening":
		activityType = discordgo.ActivityTypeListening
	case "competing":
		activityType = discordgo.ActivityTypeCompeting
	}

	if err := yuno.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{{
			Name: bot.Global.Bot.Status,
			Type: activityType,
		}},
		Status: "online",
	}); err != nil {
		log.Printf("Warning: Could not set initial status: %v", err)
	}

	// 4. Open websocket connection
	log.Printf("→ Yuno is starting... | Prefix: %s | Owner(s): %v", bot.Global.Bot.Prefix, bot.Global.Bot.OwnerIDs)
	if err := yuno.Session.Open(); err != nil {
		log.Fatalf("Cannot open Discord session: %v", err)
	}
	defer yuno.Stop()

	// 5. Ready message
	log.Println("┌────────────────────────────────────────┐")
	log.Printf("│  Yuno is online → %s", yuno.Session.State.User.String())
	log.Printf("│  Serving %d guild(s) | discordgo %s", len(yuno.Session.State.Guilds), discordgo.VERSION)
	log.Println("└────────────────────────────────────────┘")
	log.Println("   Press Ctrl+C to shut down gracefully ♡")

	// 6. Graceful shutdown handling
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// 7. Farewell
	log.Println("→ Shutting down... Ara ara~")
	time.Sleep(500 * time.Millisecond)
}
