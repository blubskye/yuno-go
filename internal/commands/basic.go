package commands

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PingCommand responds with pong
type PingCommand struct{}

func (c *PingCommand) Name() string        { return "ping" }
func (c *PingCommand) Aliases() []string   { return []string{"pong"} }
func (c *PingCommand) Description() string { return "Check if bot is responsive" }
func (c *PingCommand) Usage() string       { return "ping" }
func (c *PingCommand) RequiredPermissions() []int64 { return nil }
func (c *PingCommand) MasterOnly() bool    { return false }

func (c *PingCommand) Execute(ctx *Context) error {
	msg := "🏓 Pong!"
	
	if ctx.Message != nil {
		// Calculate latency
		timestamp := ctx.Message.Timestamp
		latency := time.Since(timestamp)
		msg = fmt.Sprintf("🏓 Pong! Latency: %dms", latency.Milliseconds())
		
		_, err := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, msg)
		return err
	}
	
	fmt.Println(msg)
	return nil
}

// StatsCommand shows bot statistics
type StatsCommand struct{}

func (c *StatsCommand) Name() string        { return "stats" }
func (c *StatsCommand) Aliases() []string   { return []string{"info", "status"} }
func (c *StatsCommand) Description() string { return "Show bot statistics" }
func (c *StatsCommand) Usage() string       { return "stats" }
func (c *StatsCommand) RequiredPermissions() []int64 { return nil }
func (c *StatsCommand) MasterOnly() bool    { return false }

func (c *StatsCommand) Execute(ctx *Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	embed := &discordgo.MessageEmbed{
		Title: "Yuno Gasai Statistics",
		Color: 0xFF51FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Memory Usage",
				Value:  fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
				Inline: true,
			},
			{
				Name:   "Goroutines",
				Value:  fmt.Sprintf("%d", runtime.NumGoroutine()),
				Inline: true,
			},
			{
				Name:   "Go Version",
				Value:  runtime.Version(),
				Inline: true,
			},
		},
	}
	
	if ctx.Message != nil {
		// Add guild count
		guilds := len(ctx.Session.State.Guilds)
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Guilds",
			Value:  fmt.Sprintf("%d", guilds),
			Inline: true,
		})
		
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}
	
	// Terminal output
	fmt.Printf("Memory: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("Go Version: %s\n", runtime.Version())
	return nil
}

// ShutdownCommand gracefully shuts down the bot
type ShutdownCommand struct{}

func (c *ShutdownCommand) Name() string        { return "shutdown" }
func (c *ShutdownCommand) Aliases() []string   { return []string{"stop"} }
func (c *ShutdownCommand) Description() string { return "Shutdown the bot" }
func (c *ShutdownCommand) Usage() string       { return "shutdown" }
func (c *ShutdownCommand) RequiredPermissions() []int64 { return nil }
func (c *ShutdownCommand) MasterOnly() bool    { return true }

func (c *ShutdownCommand) Execute(ctx *Context) error {
	if ctx.Message != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Shutting down... 👋")
	}
	
	fmt.Println("Shutdown command received")
	// The actual shutdown is handled by main.go signal handling
	// This just acknowledges the command
	return nil
}