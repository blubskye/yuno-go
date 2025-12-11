package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// UrbanDictionaryResponse represents the Urban Dictionary API response
type UrbanDictionaryResponse struct {
	List []UrbanDefinition `json:"list"`
}

// UrbanDefinition represents a single Urban Dictionary definition
type UrbanDefinition struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
	Example    string `json:"example"`
	ThumbsUp   int    `json:"thumbs_up"`
	ThumbsDown int    `json:"thumbs_down"`
	Permalink  string `json:"permalink"`
}

// NekosLifeResponse represents a response from nekos.life API
type NekosLifeResponse struct {
	URL string `json:"url"`
}

var praiseMessages = []string{
	"%s, you're doing amazing! Keep it up!",
	"Great job, %s! Yuno is proud of you!",
	"%s is absolutely wonderful!",
	"You're the best, %s!",
	"%s, you make everything better!",
	"Ara ara~ %s, you're so talented!",
	"%s deserves all the headpats!",
	"Good job, %s! You're a star!",
	"%s is precious and must be protected!",
	"Yuno believes in you, %s!",
}

var scoldMessages = []string{
	"%s, you disappointed Yuno...",
	"Bad %s! No headpats for you!",
	"%s, Yuno is watching... and judging.",
	"Tsk tsk, %s. Do better.",
	"%s, you made Yuno sad...",
	"Ara ara~ %s, that wasn't very nice.",
	"%s needs to reflect on their actions!",
	"Yuno is not happy with you, %s!",
	"%s, consider this your final warning!",
	"Bad! Bad %s!",
}

var eightBallResponses = []string{
	"It is certain.",
	"It is decidedly so.",
	"Without a doubt.",
	"Yes definitely.",
	"You may rely on it.",
	"As I see it, yes.",
	"Most likely.",
	"Outlook good.",
	"Yes.",
	"Signs point to yes.",
	"Reply hazy, try again.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	"Don't count on it.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Very doubtful.",
}

// UrbanCommand looks up terms on Urban Dictionary
type UrbanCommand struct{}

func (c *UrbanCommand) Name() string        { return "urban" }
func (c *UrbanCommand) Aliases() []string   { return []string{"ud", "define"} }
func (c *UrbanCommand) Description() string { return "Look up a term on Urban Dictionary" }
func (c *UrbanCommand) Usage() string       { return "urban <term>" }
func (c *UrbanCommand) RequiredPermissions() []int64 { return nil }
func (c *UrbanCommand) MasterOnly() bool    { return false }

func (c *UrbanCommand) Execute(ctx *Context) error {
	if len(ctx.Args) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: `"+ctx.GetPrefix()+"urban <term>`")
		}
		return nil
	}

	term := strings.Join(ctx.Args, " ")
	searchURL := fmt.Sprintf("https://api.urbandictionary.com/v0/define?term=%s", url.QueryEscape(term))

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error connecting to Urban Dictionary.")
		}
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result UrbanDictionaryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.List) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("No definition found for `%s`", term))
		}
		return nil
	}

	def := result.List[0]

	// Clean up Urban Dictionary formatting (remove brackets)
	definition := strings.ReplaceAll(def.Definition, "[", "")
	definition = strings.ReplaceAll(definition, "]", "")
	example := strings.ReplaceAll(def.Example, "[", "")
	example = strings.ReplaceAll(example, "]", "")

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Urban Dictionary: %s", def.Word),
		URL:   def.Permalink,
		Color: 0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Definition",
				Value:  truncate(definition, 1024),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("👍 %d | 👎 %d", def.ThumbsUp, def.ThumbsDown),
		},
	}

	if example != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Example",
			Value:  "*" + truncate(example, 1024) + "*",
			Inline: false,
		})
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	fmt.Printf("Urban: %s\n%s\n", def.Word, definition)
	return nil
}

// PraiseCommand praises someone
type PraiseCommand struct{}

func (c *PraiseCommand) Name() string        { return "praise" }
func (c *PraiseCommand) Aliases() []string   { return nil }
func (c *PraiseCommand) Description() string { return "Praise someone" }
func (c *PraiseCommand) Usage() string       { return "praise [@user]" }
func (c *PraiseCommand) RequiredPermissions() []int64 { return nil }
func (c *PraiseCommand) MasterOnly() bool    { return false }

func (c *PraiseCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	var targetName string
	if len(ctx.Message.Mentions) > 0 {
		targetName = ctx.Message.Mentions[0].Username
	} else if len(ctx.Args) > 0 {
		targetName = strings.Join(ctx.Args, " ")
	} else {
		targetName = ctx.Message.Author.Username
	}

	message := fmt.Sprintf(praiseMessages[rand.Intn(len(praiseMessages))], targetName)

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0x00FF00,
	}

	// Try to get a pat gif
	resp, err := httpClient.Get("https://nekos.life/api/v2/img/pat")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var nekos NekosLifeResponse
			if json.Unmarshal(body, &nekos) == nil && nekos.URL != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: nekos.URL}
			}
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ScoldCommand scolds someone
type ScoldCommand struct{}

func (c *ScoldCommand) Name() string        { return "scold" }
func (c *ScoldCommand) Aliases() []string   { return nil }
func (c *ScoldCommand) Description() string { return "Scold someone" }
func (c *ScoldCommand) Usage() string       { return "scold <@user>" }
func (c *ScoldCommand) RequiredPermissions() []int64 { return nil }
func (c *ScoldCommand) MasterOnly() bool    { return false }

func (c *ScoldCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Message.Mentions) == 0 && len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "You need to specify someone to scold!")
		return nil
	}

	var targetName string
	if len(ctx.Message.Mentions) > 0 {
		targetName = ctx.Message.Mentions[0].Username
	} else {
		targetName = strings.Join(ctx.Args, " ")
	}

	message := fmt.Sprintf(scoldMessages[rand.Intn(len(scoldMessages))], targetName)

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0xFF0000,
	}

	// Try to get a pout gif
	resp, err := httpClient.Get("https://nekos.life/api/v2/img/pout")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var nekos NekosLifeResponse
			if json.Unmarshal(body, &nekos) == nil && nekos.URL != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: nekos.URL}
			}
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// EightBallCommand magic 8-ball
type EightBallCommand struct{}

func (c *EightBallCommand) Name() string        { return "8ball" }
func (c *EightBallCommand) Aliases() []string   { return []string{"eightball", "magic8ball"} }
func (c *EightBallCommand) Description() string { return "Ask the magic 8-ball a question" }
func (c *EightBallCommand) Usage() string       { return "8ball <question>" }
func (c *EightBallCommand) RequiredPermissions() []int64 { return nil }
func (c *EightBallCommand) MasterOnly() bool    { return false }

func (c *EightBallCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "You need to ask a question!")
		return nil
	}

	question := strings.Join(ctx.Args, " ")
	answer := eightBallResponses[rand.Intn(len(eightBallResponses))]

	embed := &discordgo.MessageEmbed{
		Title: "🎱 Magic 8-Ball",
		Color: 0x9B59B6,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Question",
				Value:  question,
				Inline: false,
			},
			{
				Name:   "Answer",
				Value:  answer,
				Inline: false,
			},
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// HugCommand sends a hug gif
type HugCommand struct{}

func (c *HugCommand) Name() string        { return "hug" }
func (c *HugCommand) Aliases() []string   { return nil }
func (c *HugCommand) Description() string { return "Hug someone" }
func (c *HugCommand) Usage() string       { return "hug <@user>" }
func (c *HugCommand) RequiredPermissions() []int64 { return nil }
func (c *HugCommand) MasterOnly() bool    { return false }

func (c *HugCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Message.Mentions) == 0 && len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Who do you want to hug?")
		return nil
	}

	var targetName string
	if len(ctx.Message.Mentions) > 0 {
		targetName = ctx.Message.Mentions[0].Username
	} else {
		targetName = strings.Join(ctx.Args, " ")
	}

	message := fmt.Sprintf("%s hugs %s!", ctx.Message.Author.Username, targetName)

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0xFF003D,
	}

	resp, err := httpClient.Get("https://nekos.life/api/v2/img/hug")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var nekos NekosLifeResponse
			if json.Unmarshal(body, &nekos) == nil && nekos.URL != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: nekos.URL}
			}
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// SlapCommand sends a slap gif
type SlapCommand struct{}

func (c *SlapCommand) Name() string        { return "slap" }
func (c *SlapCommand) Aliases() []string   { return nil }
func (c *SlapCommand) Description() string { return "Slap someone" }
func (c *SlapCommand) Usage() string       { return "slap <@user>" }
func (c *SlapCommand) RequiredPermissions() []int64 { return nil }
func (c *SlapCommand) MasterOnly() bool    { return false }

func (c *SlapCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Message.Mentions) == 0 && len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Who do you want to slap?")
		return nil
	}

	var targetName string
	if len(ctx.Message.Mentions) > 0 {
		targetName = ctx.Message.Mentions[0].Username
	} else {
		targetName = strings.Join(ctx.Args, " ")
	}

	message := fmt.Sprintf("%s slaps %s!", ctx.Message.Author.Username, targetName)

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0xFFAA00,
	}

	resp, err := httpClient.Get("https://nekos.life/api/v2/img/slap")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var nekos NekosLifeResponse
			if json.Unmarshal(body, &nekos) == nil && nekos.URL != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: nekos.URL}
			}
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// KissCommand sends a kiss gif
type KissCommand struct{}

func (c *KissCommand) Name() string        { return "kiss" }
func (c *KissCommand) Aliases() []string   { return nil }
func (c *KissCommand) Description() string { return "Kiss someone" }
func (c *KissCommand) Usage() string       { return "kiss <@user>" }
func (c *KissCommand) RequiredPermissions() []int64 { return nil }
func (c *KissCommand) MasterOnly() bool    { return false }

func (c *KissCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	if len(ctx.Message.Mentions) == 0 && len(ctx.Args) == 0 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Who do you want to kiss?")
		return nil
	}

	var targetName string
	if len(ctx.Message.Mentions) > 0 {
		targetName = ctx.Message.Mentions[0].Username
	} else {
		targetName = strings.Join(ctx.Args, " ")
	}

	message := fmt.Sprintf("%s kisses %s! 💕", ctx.Message.Author.Username, targetName)

	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       0xFF003D,
	}

	resp, err := httpClient.Get("https://nekos.life/api/v2/img/kiss")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var nekos NekosLifeResponse
			if json.Unmarshal(body, &nekos) == nil && nekos.URL != "" {
				embed.Image = &discordgo.MessageEmbedImage{URL: nekos.URL}
			}
		}
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// NekoCommand gets a random catgirl image
type NekoCommand struct{}

func (c *NekoCommand) Name() string        { return "neko" }
func (c *NekoCommand) Aliases() []string   { return []string{"catgirl"} }
func (c *NekoCommand) Description() string { return "Get a random catgirl image" }
func (c *NekoCommand) Usage() string       { return "neko" }
func (c *NekoCommand) RequiredPermissions() []int64 { return nil }
func (c *NekoCommand) MasterOnly() bool    { return false }

func (c *NekoCommand) Execute(ctx *Context) error {
	if ctx.Message == nil {
		return nil
	}

	resp, err := httpClient.Get("https://nekos.life/api/v2/img/neko")
	if err != nil {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to get neko image.")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Failed to get neko image.")
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var nekos NekosLifeResponse
	if err := json.Unmarshal(body, &nekos); err != nil {
		return err
	}

	if nekos.URL == "" {
		ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "No image found.")
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title: "🐱 Neko!",
		Color: 0xFF003D,
		Image: &discordgo.MessageEmbedImage{URL: nekos.URL},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
