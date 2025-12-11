/*
    Yuno Gasai. A Discord.JS based bot, with multiple features.
    Copyright (C) 2018 Maeeen <maeeennn@gmail.com>

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see https://www.gnu.org/licenses/.
*/

package bot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Terminal handles interactive terminal commands
type Terminal struct {
	bot          *Bot
	running      bool
	watchers     map[string]chan struct{} // Channel ID -> stop channel
	watchersMu   sync.Mutex
}

// NewTerminal creates a new terminal handler
func NewTerminal(b *Bot) *Terminal {
	return &Terminal{
		bot:      b,
		running:  false,
		watchers: make(map[string]chan struct{}),
	}
}

// Start begins the terminal input loop
func (t *Terminal) Start() {
	t.running = true
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n💻 Terminal ready. Type 'help' for available commands.")
	fmt.Println("────────────────────────────────────────────────────────")

	for t.running {
		fmt.Print("yuno> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		t.handleCommand(input)
	}
}

// Stop stops the terminal
func (t *Terminal) Stop() {
	t.running = false
	// Stop all watchers
	t.watchersMu.Lock()
	for _, ch := range t.watchers {
		close(ch)
	}
	t.watchers = make(map[string]chan struct{})
	t.watchersMu.Unlock()
}

// handleCommand processes terminal commands
func (t *Terminal) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "help", "?":
		t.showHelp()
	case "servers", "guilds":
		t.listServers(args)
	case "channels":
		t.listChannels(args)
	case "send":
		t.sendMessage(args)
	case "messages", "msgs":
		t.fetchMessages(args)
	case "watch":
		t.watchChannel(args)
	case "inbox":
		t.showInbox(args)
	case "reply":
		t.replyToDM(args)
	case "tban":
		t.terminalBan(args)
	case "texportbans":
		t.exportBans(args)
	case "timportbans":
		t.importBans(args)
	case "bot-ban":
		t.botBan(args)
	case "bot-unban":
		t.botUnban(args)
	case "bot-banlist":
		t.botBanlist(args)
	case "set-presence", "presence":
		t.setPresence(args)
	case "status":
		t.showStatus()
	case "quit", "exit":
		fmt.Println("Use Ctrl+C to shutdown the bot gracefully.")
	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
	}
}

func (t *Terminal) showHelp() {
	fmt.Println(`
💻 Terminal Commands
────────────────────────────────────────────────────────

Server & Channel Management:
  servers [-v]              List all servers (verbose mode)
  channels <server-id>      List channels in a server

Message Commands:
  send <channel-id> <msg>   Send a message to a channel
  messages <channel-id> [n] Fetch last N messages (default: 20)
  watch <channel-id>        Watch a channel in real-time
  watch stop <channel-id>   Stop watching a channel
  watch stop all            Stop all watchers

DM Inbox:
  inbox [n]                 View inbox (default: 10 messages)
  inbox user <user-id>      View DMs from specific user
  inbox unread              Count unread DMs
  reply <id> <message>      Reply to a DM by inbox ID or user ID

Ban Commands:
  tban <server> <user> [r]  Ban user from server
  texportbans <server>      Export server bans to file
  timportbans <server> <f>  Import bans from file

Bot-Level Bans:
  bot-ban <user|server> <id> [reason]  Ban from using bot
  bot-unban <id>                       Remove bot ban
  bot-banlist [users|servers]          List bot bans

Bot Control:
  set-presence <type> <text>   Set bot activity
  set-presence status <s>      Set online status
  set-presence clear           Clear activity
  status                       Show bot status
  help                         Show this help
────────────────────────────────────────────────────────`)
}

func (t *Terminal) listServers(args []string) {
	verbose := len(args) > 0 && (args[0] == "-v" || args[0] == "--verbose")

	guilds := t.bot.Session.State.Guilds
	if len(guilds) == 0 {
		fmt.Println("No servers found.")
		return
	}

	fmt.Printf("\n🏰 Servers (%d)\n", len(guilds))
	fmt.Println("────────────────────────────────────────")

	for _, g := range guilds {
		if verbose {
			fmt.Printf("  %s\n", g.Name)
			fmt.Printf("    ID: %s\n", g.ID)
			fmt.Printf("    Members: %d\n", g.MemberCount)
			fmt.Printf("    Owner: %s\n", g.OwnerID)
			fmt.Println()
		} else {
			fmt.Printf("  [%s] %s (%d members)\n", g.ID, g.Name, g.MemberCount)
		}
	}
}

func (t *Terminal) listChannels(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: channels <server-id|server-name>")
		return
	}

	guildID := args[0]

	// Try to find by name if not an ID
	guild, err := t.bot.Session.State.Guild(guildID)
	if err != nil {
		// Search by name
		for _, g := range t.bot.Session.State.Guilds {
			if strings.EqualFold(g.Name, guildID) || strings.Contains(strings.ToLower(g.Name), strings.ToLower(guildID)) {
				guild = g
				break
			}
		}
	}

	if guild == nil {
		fmt.Println("Server not found.")
		return
	}

	channels, err := t.bot.Session.GuildChannels(guild.ID)
	if err != nil {
		fmt.Printf("Error fetching channels: %v\n", err)
		return
	}

	// Sort by position
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Position < channels[j].Position
	})

	fmt.Printf("\n📺 Channels in %s\n", guild.Name)
	fmt.Println("────────────────────────────────────────")

	// Group by category
	categories := make(map[string][]*discordgo.Channel)
	var noCategory []*discordgo.Channel

	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory {
			continue
		}
		if ch.ParentID == "" {
			noCategory = append(noCategory, ch)
		} else {
			categories[ch.ParentID] = append(categories[ch.ParentID], ch)
		}
	}

	// Print channels without category
	for _, ch := range noCategory {
		printChannel(ch)
	}

	// Print categories and their channels
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory {
			fmt.Printf("\n📁 %s\n", ch.Name)
			for _, subCh := range categories[ch.ID] {
				printChannel(subCh)
			}
		}
	}
}

func printChannel(ch *discordgo.Channel) {
	icon := "💬"
	switch ch.Type {
	case discordgo.ChannelTypeGuildVoice:
		icon = "🔊"
	case discordgo.ChannelTypeGuildNews:
		icon = "📢"
	case discordgo.ChannelTypeGuildStageVoice:
		icon = "🎭"
	case discordgo.ChannelTypeGuildForum:
		icon = "📋"
	}
	fmt.Printf("  %s [%s] %s\n", icon, ch.ID, ch.Name)
}

func (t *Terminal) sendMessage(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: send <channel-id> <message>")
		return
	}

	channelID := args[0]
	message := strings.Join(args[1:], " ")

	_, err := t.bot.Session.ChannelMessageSend(channelID, message)
	if err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		return
	}

	fmt.Printf("✅ Message sent to %s\n", channelID)
}

func (t *Terminal) fetchMessages(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: messages <channel-id> [count]")
		return
	}

	channelID := args[0]
	limit := 20
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil && n > 0 {
			limit = n
		}
	}

	messages, err := t.bot.Session.ChannelMessages(channelID, limit, "", "", "")
	if err != nil {
		fmt.Printf("Error fetching messages: %v\n", err)
		return
	}

	channel, _ := t.bot.Session.Channel(channelID)
	channelName := channelID
	if channel != nil {
		channelName = channel.Name
	}

	fmt.Printf("\n📜 Messages in #%s (last %d)\n", channelName, len(messages))
	fmt.Println("────────────────────────────────────────")

	// Reverse to show oldest first
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		ts := time.Time(m.Timestamp)
		timeStr := ts.Format("15:04:05")
		content := m.Content
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		fmt.Printf("[%s] %s: %s\n", timeStr, m.Author.Username, content)
	}
}

func (t *Terminal) watchChannel(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: watch <channel-id> | watch stop <channel-id|all>")
		return
	}

	if args[0] == "stop" {
		if len(args) < 2 {
			fmt.Println("Usage: watch stop <channel-id|all>")
			return
		}

		t.watchersMu.Lock()
		defer t.watchersMu.Unlock()

		if args[1] == "all" {
			for id, ch := range t.watchers {
				close(ch)
				delete(t.watchers, id)
			}
			fmt.Println("✅ Stopped all watchers")
			return
		}

		if ch, ok := t.watchers[args[1]]; ok {
			close(ch)
			delete(t.watchers, args[1])
			fmt.Printf("✅ Stopped watching %s\n", args[1])
		} else {
			fmt.Printf("Not watching channel %s\n", args[1])
		}
		return
	}

	channelID := args[0]

	t.watchersMu.Lock()
	if _, ok := t.watchers[channelID]; ok {
		t.watchersMu.Unlock()
		fmt.Printf("Already watching channel %s\n", channelID)
		return
	}

	stopCh := make(chan struct{})
	t.watchers[channelID] = stopCh
	t.watchersMu.Unlock()

	channel, _ := t.bot.Session.Channel(channelID)
	channelName := channelID
	if channel != nil {
		channelName = "#" + channel.Name
	}

	fmt.Printf("👁️ Now watching %s (use 'watch stop %s' to stop)\n", channelName, channelID)

	// Add message handler
	handler := t.bot.Session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.ChannelID == channelID {
			ts := time.Time(m.Timestamp)
			timeStr := ts.Format("15:04:05")
			fmt.Printf("\n[%s] [%s] %s: %s\nyuno> ", timeStr, channelName, m.Author.Username, m.Content)
		}
	})

	// Wait for stop signal in background
	go func() {
		<-stopCh
		handler()
	}()
}

func (t *Terminal) showInbox(args []string) {
	limit := 10

	if len(args) > 0 {
		switch args[0] {
		case "unread":
			count, err := t.bot.DB.GetUnreadDMCount()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			fmt.Printf("📬 %d unread DM(s)\n", count)
			return
		case "user":
			if len(args) < 2 {
				fmt.Println("Usage: inbox user <user-id>")
				return
			}
			dms, err := t.bot.DB.GetDMsByUser(args[1], 20)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			t.printDMs(dms)
			return
		default:
			if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
				limit = n
			}
		}
	}

	dms, err := t.bot.DB.GetDMs(limit)
	if err != nil {
		fmt.Printf("Error fetching inbox: %v\n", err)
		return
	}

	t.printDMs(dms)
}

func (t *Terminal) printDMs(dms []map[string]interface{}) {
	if len(dms) == 0 {
		fmt.Println("📬 Inbox is empty.")
		return
	}

	fmt.Printf("\n📬 Inbox (%d messages)\n", len(dms))
	fmt.Println("────────────────────────────────────────")

	for _, dm := range dms {
		id := dm["id"].(int64)
		userTag := dm["user_tag"].(string)
		content := dm["content"].(string)
		receivedAt := dm["received_at"].(string)
		read := dm["read"].(bool)

		readIcon := "📩"
		if read {
			readIcon = "📭"
		}

		if len(content) > 60 {
			content = content[:57] + "..."
		}

		fmt.Printf("%s [#%d] %s (%s)\n", readIcon, id, userTag, receivedAt)
		fmt.Printf("   %s\n", content)
	}
}

func (t *Terminal) replyToDM(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: reply <inbox-id|user-id> <message>")
		return
	}

	target := args[0]
	message := strings.Join(args[1:], " ")

	var userID string

	// Check if it's an inbox ID (small number) or user ID (snowflake)
	if id, err := strconv.ParseInt(target, 10, 64); err == nil && id < 1000000 {
		// It's an inbox ID
		dm, err := t.bot.DB.GetDMByID(id)
		if err != nil {
			fmt.Printf("Error finding DM #%d: %v\n", id, err)
			return
		}
		userID = dm["user_id"].(string)
		t.bot.DB.MarkDMRead(id)
	} else {
		// It's a user ID
		userID = target
	}

	// Create DM channel
	channel, err := t.bot.Session.UserChannelCreate(userID)
	if err != nil {
		fmt.Printf("Error creating DM channel: %v\n", err)
		return
	}

	_, err = t.bot.Session.ChannelMessageSend(channel.ID, message)
	if err != nil {
		fmt.Printf("Error sending reply: %v\n", err)
		return
	}

	fmt.Printf("✅ Reply sent to %s\n", userID)
}

func (t *Terminal) terminalBan(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: tban <server-id> <user-id> [reason]")
		return
	}

	guildID := args[0]
	userID := args[1]
	reason := "Banned via terminal"
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	err := t.bot.Session.GuildBanCreateWithReason(guildID, userID, reason, 0)
	if err != nil {
		fmt.Printf("Error banning user: %v\n", err)
		return
	}

	fmt.Printf("✅ Banned user %s from server %s\n", userID, guildID)
}

func (t *Terminal) exportBans(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: texportbans <server-id> [output-file]")
		return
	}

	guildID := args[0]
	outputFile := fmt.Sprintf("BANS-%s-%s.json", guildID, time.Now().Format("20060102-150405"))
	if len(args) > 1 {
		outputFile = args[1]
	}

	bans, err := t.bot.Session.GuildBans(guildID, 0, "", "")
	if err != nil {
		fmt.Printf("Error fetching bans: %v\n", err)
		return
	}

	if len(bans) == 0 {
		fmt.Println("No bans found in this server.")
		return
	}

	// Export format
	type BanExport struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Reason   string `json:"reason"`
	}

	var exports []BanExport
	for _, ban := range bans {
		exports = append(exports, BanExport{
			UserID:   ban.User.ID,
			Username: ban.User.String(),
			Reason:   ban.Reason,
		})
	}

	data, err := json.MarshalIndent(exports, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling bans: %v\n", err)
		return
	}

	err = os.WriteFile(outputFile, data, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Printf("✅ Exported %d bans to %s\n", len(exports), outputFile)
}

func (t *Terminal) importBans(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: timportbans <server-id> <input-file>")
		return
	}

	guildID := args[0]
	inputFile := args[1]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Try JSON format first
	type BanImport struct {
		UserID string `json:"user_id"`
		Reason string `json:"reason"`
	}

	var imports []BanImport
	if err := json.Unmarshal(data, &imports); err != nil {
		// Try plain text format (one ID per line)
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				imports = append(imports, BanImport{UserID: line, Reason: "Imported ban"})
			}
		}
	}

	if len(imports) == 0 {
		fmt.Println("No bans to import.")
		return
	}

	fmt.Printf("Importing %d bans...\n", len(imports))

	success := 0
	for _, ban := range imports {
		reason := ban.Reason
		if reason == "" {
			reason = "Imported ban"
		}
		err := t.bot.Session.GuildBanCreateWithReason(guildID, ban.UserID, reason, 0)
		if err != nil {
			fmt.Printf("  ❌ Failed to ban %s: %v\n", ban.UserID, err)
		} else {
			success++
		}
	}

	fmt.Printf("✅ Imported %d/%d bans\n", success, len(imports))
}

func (t *Terminal) botBan(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bot-ban <user|server> <id> [reason]")
		return
	}

	banType := strings.ToLower(args[0])
	if banType != "user" && banType != "server" {
		fmt.Println("Invalid type. Use 'user' or 'server'.")
		return
	}

	targetID := args[1]
	reason := "Banned via terminal"
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	err := t.bot.DB.AddBotBan(targetID, banType, reason, "terminal")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Bot-banned %s %s\n", banType, targetID)
}

func (t *Terminal) botUnban(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bot-unban <id>")
		return
	}

	err := t.bot.DB.RemoveBotBan(args[0])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Removed bot-ban for %s\n", args[0])
}

func (t *Terminal) botBanlist(args []string) {
	filterType := ""
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "users", "user":
			filterType = "user"
		case "servers", "server":
			filterType = "server"
		}
	}

	bans, err := t.bot.DB.GetBotBans(filterType)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(bans) == 0 {
		fmt.Println("📋 No bot-level bans.")
		return
	}

	fmt.Printf("\n🚫 Bot-Level Bans (%d)\n", len(bans))
	fmt.Println("────────────────────────────────────────")

	for _, ban := range bans {
		icon := "👤"
		if ban["type"] == "server" {
			icon = "🏠"
		}
		fmt.Printf("%s [%s] %s - %s\n", icon, ban["type"], ban["id"], ban["reason"])
	}
}

func (t *Terminal) setPresence(args []string) {
	if len(args) < 1 {
		fmt.Println(`Set Presence Command

Usage:
  set-presence <type> <text>     - Set activity
  set-presence status <status>   - Set online status
  set-presence clear             - Clear activity

Activity Types:
  playing, watching, listening, streaming, competing

Status Options:
  online, idle, dnd, invisible

Examples:
  set-presence playing with Yukki
  set-presence watching over senpai
  set-presence status dnd
  set-presence clear`)
		return
	}

	subcommand := strings.ToLower(args[0])

	// Handle clear
	if subcommand == "clear" || subcommand == "reset" || subcommand == "none" {
		err := t.bot.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Activities: []*discordgo.Activity{},
			Status:     "online",
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("✅ Presence cleared")
		return
	}

	// Handle status change
	if subcommand == "status" {
		if len(args) < 2 {
			fmt.Println("Usage: set-presence status <online|idle|dnd|invisible>")
			return
		}
		status := strings.ToLower(args[1])
		validStatuses := map[string]bool{"online": true, "idle": true, "dnd": true, "invisible": true}
		if !validStatuses[status] {
			fmt.Println("Invalid status. Options: online, idle, dnd, invisible")
			return
		}

		err := t.bot.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: status,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("✅ Status set to %s\n", status)
		return
	}

	// Handle activity type
	activityTypes := map[string]discordgo.ActivityType{
		"playing":   discordgo.ActivityTypeGame,
		"streaming": discordgo.ActivityTypeStreaming,
		"listening": discordgo.ActivityTypeListening,
		"watching":  discordgo.ActivityTypeWatching,
		"competing": discordgo.ActivityTypeCompeting,
	}

	activityType, ok := activityTypes[subcommand]
	if !ok {
		fmt.Printf("Unknown type: %s\nValid types: playing, watching, listening, streaming, competing\n", subcommand)
		return
	}

	if len(args) < 2 {
		fmt.Println("Please provide activity text!")
		return
	}

	activityText := strings.Join(args[1:], " ")

	err := t.bot.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{{
			Name: activityText,
			Type: activityType,
		}},
		Status: "online",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Now %s %s\n", subcommand, activityText)
}

func (t *Terminal) showStatus() {
	s := t.bot.Session

	fmt.Println("\n📊 Bot Status")
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("User: %s\n", s.State.User.String())
	fmt.Printf("Guilds: %d\n", len(s.State.Guilds))

	unread, _ := t.bot.DB.GetUnreadDMCount()
	fmt.Printf("Unread DMs: %d\n", unread)

	bans, _ := t.bot.DB.GetBotBans("")
	fmt.Printf("Bot Bans: %d\n", len(bans))
}
