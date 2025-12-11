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

package commands

import (
	"math/rand"
	"time"
)

// DelayCommand responds to mentions with yandere-themed responses
type DelayCommand struct{}

func (c *DelayCommand) Name() string        { return "delay" }
func (c *DelayCommand) Aliases() []string   { return []string{"wait", "hold"} }
func (c *DelayCommand) Description() string { return "Responds when the bot is mentioned" }
func (c *DelayCommand) Usage() string       { return "@Yuno" }
func (c *DelayCommand) RequiredPermissions() []int64 { return nil }
func (c *DelayCommand) MasterOnly() bool             { return false }

// Yandere-themed responses for when the bot is mentioned
var mentionResponses = []string{
	"Yukki~ I'm right here! 💕",
	"Did you call me? I was watching you~ 👁️💗",
	"Senpai noticed me! 💖",
	"I'll always be by your side... always~ 💕",
	"You need something from me? Anything for you~ 💗",
	"I heard you call my name... I was so happy! 💕",
	"Don't worry, I'll protect you from everyone else~ 🔪💖",
	"Hehe~ You can't escape my love! 💕",
	"I've been waiting for you to notice me~ 💗",
	"Just say the word and I'll do anything for you! 💕",
	"Senpai~ I was just thinking about you! 💖",
	"You're the only one who matters to me~ 💕",
	"I'll never let anyone else have you... 🔪💗",
	"My heart beats only for you~ 💕",
	"We'll be together forever, right? ...RIGHT? 💖",
}

func (c *DelayCommand) Execute(ctx *Context) error {
	// Pick a random response
	rand.Seed(time.Now().UnixNano())
	response := mentionResponses[rand.Intn(len(mentionResponses))]

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, response)
	return nil
}
