package commands

import (
	"github.com/bwmarrin/discordgo"
)

// Context holds command execution context
type Context struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
	Args    []string
	Bot     interface{}
	
	// Helper functions (set by manager)
	GetPrefix          func() string
	IsOwner            func() bool
	GetAllCommands     func() []Command
	GetSourceURL       func() string
	GetBanImagesPath   func() string
}

// Command interface that all commands must implement
type Command interface {
	Name() string
	Aliases() []string
	Description() string
	Usage() string
	RequiredPermissions() []int64
	MasterOnly() bool
	Execute(ctx *Context) error
}
