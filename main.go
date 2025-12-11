package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"yuno-go/internal/bot"
)

// Connection state tracking
var (
	isConnected     atomic.Bool
	reconnectCount  atomic.Int32
	lastDisconnect  atomic.Int64
)

// displayStartupBanner shows the ASCII art banner
func displayStartupBanner() {
	// Read ASCII art from file
	artBytes, err := os.ReadFile("ascii.txt")
	if err != nil {
		// Fallback if file doesn't exist
		fmt.Println("\n╔══════════════════════════════════════╗")
		fmt.Println("║             YUNO BOT GO              ║")
		fmt.Println("╚══════════════════════════════════════╝\n")
		return
	}

	// Display "YUNO" header
	fmt.Println()
	fmt.Println(string(artBytes))
	fmt.Println()
}

// Command-line flags
var (
	debugFlag   = flag.Bool("debug", false, "Enable debug mode with verbose logging")
	traceFlag   = flag.Bool("trace", false, "Enable full stack traces on panics")
	configPath  = flag.String("config", "config.toml", "Path to configuration file")
)

// setupReconnectionHandlers adds handlers for connection events
func setupReconnectionHandlers(s *discordgo.Session) {
	// Handle successful connection
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		isConnected.Store(true)
		count := reconnectCount.Load()
		if count > 0 {
			log.Printf("✓ Reconnected successfully (attempt #%d)", count)
			reconnectCount.Store(0)
		}
	})

	// Handle disconnection
	s.AddHandler(func(s *discordgo.Session, d *discordgo.Disconnect) {
		isConnected.Store(false)
		lastDisconnect.Store(time.Now().Unix())
		log.Printf("⚠️  Disconnected from Discord gateway")
	})

	// Handle resumed connection
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Resumed) {
		isConnected.Store(true)
		log.Printf("✓ Connection resumed successfully")
		reconnectCount.Store(0)
	})

	// Handle connection errors with custom reconnection logic
	s.AddHandler(func(s *discordgo.Session, c *discordgo.Connect) {
		isConnected.Store(true)
		log.Printf("✓ Connected to Discord gateway")
	})
}

// connectionMonitor watches the connection and forces reconnection if needed
func connectionMonitor(yuno *bot.Bot, stopChan <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Check if we've been disconnected for too long
			if !isConnected.Load() {
				lastDC := lastDisconnect.Load()
				if lastDC > 0 && time.Since(time.Unix(lastDC, 0)) > 2*time.Minute {
					count := reconnectCount.Add(1)
					log.Printf("⚠️  Connection lost for >2 minutes, attempting reconnect #%d...", count)

					// Close and reopen connection
					yuno.Session.Close()
					time.Sleep(5 * time.Second)

					if err := yuno.Session.Open(); err != nil {
						log.Printf("❌ Reconnect failed: %v", err)

						// Exponential backoff
						backoff := time.Duration(min(int(count)*10, 120)) * time.Second
						log.Printf("⏳ Waiting %v before next attempt...", backoff)
						time.Sleep(backoff)
					}
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Global config will be loaded here
func main() {
	// Parse command-line flags
	flag.Parse()

	// Display startup banner
	displayStartupBanner()

	// Set up stack tracing if requested
	if *traceFlag {
		debug.SetTraceback("all")
		log.Println("🔍 Full stack tracing ENABLED")
	}

	// Set debug mode
	if *debugFlag {
		bot.SetDebugMode(true)
		log.Println("🐛 Debug mode ENABLED - Verbose logging active")
	}

	// 1. Load configuration (creates default config.toml if missing)
	bot.LoadConfig(*configPath)

	// Override debug settings from command-line if provided
	if *debugFlag {
		bot.Global.Debug.Enabled = true
		bot.Global.Debug.VerboseLogging = true
	}
	if *traceFlag {
		bot.Global.Debug.FullStackTrace = true
	}

	// Apply debug settings from config
	if bot.Global.Debug.Enabled {
		bot.SetDebugMode(true)
		if !*debugFlag {
			log.Println("🐛 Debug mode ENABLED from config")
		}
	}
	if bot.Global.Debug.FullStackTrace && !*traceFlag {
		debug.SetTraceback("all")
		log.Println("🔍 Full stack tracing ENABLED from config")
	}

	// 2. Create the bot instance
	yuno, err := bot.New()
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// 2.5 Setup reconnection handlers BEFORE opening connection
	setupReconnectionHandlers(yuno.Session)

	// Enable automatic reconnection in discordgo
	yuno.Session.ShouldReconnectOnError = true

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

	// 6. Start terminal interface in goroutine
	terminal := bot.NewTerminal(yuno)
	go terminal.Start()

	// Start DM cleanup in background
	go yuno.StartDMCleanup()

	// 6.5 Start connection monitor for auto-reconnection
	monitorStop := make(chan struct{})
	go connectionMonitor(yuno, monitorStop)
	log.Println("✓ Connection monitor started (auto-reconnect enabled)")

	// 7. Graceful shutdown handling
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// 8. Farewell
	close(monitorStop) // Stop connection monitor
	terminal.Stop()
	log.Println("→ Shutting down... Ara ara~")
	time.Sleep(500 * time.Millisecond)
}
