package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Manager handles command registration and execution
type Manager struct {
	commands       map[string]Command
	aliases        map[string]string
	prefix         string
	ownerIDs       []string
	sourceURL      string
	banImagesPath  string
}

// NewManager creates a new command manager
func NewManager(prefix string, ownerIDs []string, sourceURL string, banImagesPath string) *Manager {
	return &Manager{
		commands:      make(map[string]Command),
		aliases:       make(map[string]string),
		prefix:        prefix,
		ownerIDs:      ownerIDs,
		sourceURL:     sourceURL,
		banImagesPath: banImagesPath,
	}
}

// Register adds a command to the manager
func (m *Manager) Register(cmd Command) {
	name := strings.ToLower(cmd.Name())
	m.commands[name] = cmd

	// Register aliases
	for _, alias := range cmd.Aliases() {
		alias = strings.ToLower(alias)
		m.aliases[alias] = name
	}

	log.Printf("Registered command: %s", cmd.Name())
}

// Execute runs a command
func (m *Manager) Execute(ctx *Context, content string) error {
	// Set helper functions
	ctx.GetPrefix = func() string { return m.prefix }
	ctx.IsOwner = func() bool { return m.isOwner(ctx) }
	ctx.GetAllCommands = func() []Command { return m.GetAll() }
	ctx.GetSourceURL = func() string { return m.sourceURL }
	ctx.GetBanImagesPath = func() string { return m.banImagesPath }
	
	// Parse command and args
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return nil
	}

	cmdName := strings.ToLower(parts[0])
	ctx.Args = parts[1:]

	// Find command (check aliases first)
	if alias, ok := m.aliases[cmdName]; ok {
		cmdName = alias
	}

	cmd, ok := m.commands[cmdName]
	if !ok {
		// Command not found
		return nil
	}

	// Check if master only
	if cmd.MasterOnly() && !m.isMaster(ctx) {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, 
				"❌ This command can only be used by bot owners.")
		}
		return nil
	}

	// Check permissions
	if ctx.Message != nil && ctx.Message.GuildID != "" {
		if !m.hasPermissions(ctx, cmd.RequiredPermissions()) {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, 
				"❌ You don't have permission to use this command.")
			return nil
		}
	}

	// Execute command
	return cmd.Execute(ctx)
}

// isMaster checks if user is a bot owner
func (m *Manager) isMaster(ctx *Context) bool {
	if ctx.Message == nil {
		return true // Terminal commands
	}

	for _, ownerID := range m.ownerIDs {
		if ctx.Message.Author.ID == ownerID {
			return true
		}
	}
	return false
}

// isOwner is a helper that calls isMaster
func (m *Manager) isOwner(ctx *Context) bool {
	return m.isMaster(ctx)
}

// hasPermissions checks if user has required permissions
func (m *Manager) hasPermissions(ctx *Context, perms []int64) bool {
	if len(perms) == 0 {
		return true
	}

	if ctx.Message == nil || ctx.Message.GuildID == "" {
		return true
	}

	// Get member permissions
	userPerms, err := ctx.Session.UserChannelPermissions(
		ctx.Message.Author.ID,
		ctx.Message.ChannelID,
	)
	if err != nil {
		log.Printf("Error getting permissions: %v", err)
		return false
	}

	// Check if user is admin (admin bypasses all checks)
	if userPerms&discordgo.PermissionAdministrator != 0 {
		return true
	}

	// Check each required permission
	for _, perm := range perms {
		if userPerms&perm == 0 {
			return false
		}
	}

	return true
}

// GetAll returns all registered commands
func (m *Manager) GetAll() []Command {
	commands := make([]Command, 0, len(m.commands))
	for _, cmd := range m.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// Get returns a specific command by name
func (m *Manager) Get(name string) (Command, error) {
	name = strings.ToLower(name)
	
	// Check aliases
	if alias, ok := m.aliases[name]; ok {
		name = alias
	}

	cmd, ok := m.commands[name]
	if !ok {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	return cmd, nil
}
