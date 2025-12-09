package commands

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const jikanBaseURL = "https://api.jikan.moe/v4"

// JikanAnimeResponse represents the Jikan API response for anime
type JikanAnimeResponse struct {
	Data []JikanAnime `json:"data"`
}

// JikanAnime represents an anime from Jikan API
type JikanAnime struct {
	MalID    int    `json:"mal_id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	TitleEng string `json:"title_english"`
	Synopsis string `json:"synopsis"`
	Type     string `json:"type"`
	Episodes int    `json:"episodes"`
	Status   string `json:"status"`
	Score    float64 `json:"score"`
	Rank     int    `json:"rank"`
	Popularity int  `json:"popularity"`
	Images   struct {
		JPG struct {
			ImageURL      string `json:"image_url"`
			LargeImageURL string `json:"large_image_url"`
		} `json:"jpg"`
	} `json:"images"`
	Aired struct {
		String string `json:"string"`
	} `json:"aired"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Studios []struct {
		Name string `json:"name"`
	} `json:"studios"`
}

// JikanMangaResponse represents the Jikan API response for manga
type JikanMangaResponse struct {
	Data []JikanManga `json:"data"`
}

// JikanManga represents a manga from Jikan API
type JikanManga struct {
	MalID    int    `json:"mal_id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	TitleEng string `json:"title_english"`
	Synopsis string `json:"synopsis"`
	Type     string `json:"type"`
	Chapters int    `json:"chapters"`
	Volumes  int    `json:"volumes"`
	Status   string `json:"status"`
	Score    float64 `json:"score"`
	Rank     int    `json:"rank"`
	Images   struct {
		JPG struct {
			ImageURL      string `json:"image_url"`
			LargeImageURL string `json:"large_image_url"`
		} `json:"jpg"`
	} `json:"images"`
	Published struct {
		String string `json:"string"`
	} `json:"published"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Authors []struct {
		Name string `json:"name"`
	} `json:"authors"`
}

// JikanCharacterResponse represents the Jikan API response for characters
type JikanCharacterResponse struct {
	Data []JikanCharacter `json:"data"`
}

// JikanCharacter represents a character from Jikan API
type JikanCharacter struct {
	MalID     int    `json:"mal_id"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	NameKanji string `json:"name_kanji"`
	About     string `json:"about"`
	Favorites int    `json:"favorites"`
	Images    struct {
		JPG struct {
			ImageURL string `json:"image_url"`
		} `json:"jpg"`
	} `json:"images"`
}

// HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func truncate(s string, maxLen int) string {
	s = html.UnescapeString(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// AnimeCommand searches for anime
type AnimeCommand struct{}

func (c *AnimeCommand) Name() string        { return "anime" }
func (c *AnimeCommand) Aliases() []string   { return []string{"ani"} }
func (c *AnimeCommand) Description() string { return "Search for an anime on MyAnimeList" }
func (c *AnimeCommand) Usage() string       { return "anime <query>" }
func (c *AnimeCommand) RequiredPermissions() []int64 { return nil }
func (c *AnimeCommand) MasterOnly() bool    { return false }

func (c *AnimeCommand) Execute(ctx *Context) error {
	if len(ctx.Args) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: `"+ctx.GetPrefix()+"anime <search query>`")
		}
		return nil
	}

	query := strings.Join(ctx.Args, " ")
	searchURL := fmt.Sprintf("%s/anime?q=%s&limit=1&sfw=true", jikanBaseURL, url.QueryEscape(query))

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error connecting to MyAnimeList API.")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "API rate limited. Please try again in a moment.")
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result JikanAnimeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("No anime found for `%s`", query))
		}
		return nil
	}

	anime := result.Data[0]

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title: anime.Title,
		URL:   anime.URL,
		Color: 0xFF003D,
	}

	if anime.TitleEng != "" && anime.TitleEng != anime.Title {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "English Title",
			Value:  anime.TitleEng,
			Inline: false,
		})
	}

	if anime.Images.JPG.LargeImageURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: anime.Images.JPG.LargeImageURL,
		}
	}

	if anime.Synopsis != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Synopsis",
			Value:  truncate(anime.Synopsis, 500),
			Inline: false,
		})
	}

	if anime.Type != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Type",
			Value:  anime.Type,
			Inline: true,
		})
	}

	if anime.Episodes > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Episodes",
			Value:  fmt.Sprintf("%d", anime.Episodes),
			Inline: true,
		})
	}

	if anime.Status != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  anime.Status,
			Inline: true,
		})
	}

	if anime.Score > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Score",
			Value:  fmt.Sprintf("%.2f", anime.Score),
			Inline: true,
		})
	}

	if anime.Rank > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Rank",
			Value:  fmt.Sprintf("#%d", anime.Rank),
			Inline: true,
		})
	}

	if len(anime.Genres) > 0 {
		genreNames := make([]string, 0, 5)
		for i, g := range anime.Genres {
			if i >= 5 {
				break
			}
			genreNames = append(genreNames, g.Name)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Genres",
			Value:  strings.Join(genreNames, ", "),
			Inline: true,
		})
	}

	if len(anime.Studios) > 0 {
		studioNames := make([]string, 0, 3)
		for i, s := range anime.Studios {
			if i >= 3 {
				break
			}
			studioNames = append(studioNames, s.Name)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Studio",
			Value:  strings.Join(studioNames, ", "),
			Inline: true,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Data from MyAnimeList via Jikan API",
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Terminal output
	fmt.Printf("Anime: %s\n", anime.Title)
	fmt.Printf("URL: %s\n", anime.URL)
	fmt.Printf("Score: %.2f\n", anime.Score)
	return nil
}

// MangaCommand searches for manga
type MangaCommand struct{}

func (c *MangaCommand) Name() string        { return "manga" }
func (c *MangaCommand) Aliases() []string   { return nil }
func (c *MangaCommand) Description() string { return "Search for a manga on MyAnimeList" }
func (c *MangaCommand) Usage() string       { return "manga <query>" }
func (c *MangaCommand) RequiredPermissions() []int64 { return nil }
func (c *MangaCommand) MasterOnly() bool    { return false }

func (c *MangaCommand) Execute(ctx *Context) error {
	if len(ctx.Args) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: `"+ctx.GetPrefix()+"manga <search query>`")
		}
		return nil
	}

	query := strings.Join(ctx.Args, " ")
	searchURL := fmt.Sprintf("%s/manga?q=%s&limit=1&sfw=true", jikanBaseURL, url.QueryEscape(query))

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error connecting to MyAnimeList API.")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "API rate limited. Please try again in a moment.")
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result JikanMangaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("No manga found for `%s`", query))
		}
		return nil
	}

	manga := result.Data[0]

	embed := &discordgo.MessageEmbed{
		Title: manga.Title,
		URL:   manga.URL,
		Color: 0xFF003D,
	}

	if manga.TitleEng != "" && manga.TitleEng != manga.Title {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "English Title",
			Value:  manga.TitleEng,
			Inline: false,
		})
	}

	if manga.Images.JPG.LargeImageURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: manga.Images.JPG.LargeImageURL,
		}
	}

	if manga.Synopsis != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Synopsis",
			Value:  truncate(manga.Synopsis, 500),
			Inline: false,
		})
	}

	if manga.Type != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Type",
			Value:  manga.Type,
			Inline: true,
		})
	}

	if manga.Chapters > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Chapters",
			Value:  fmt.Sprintf("%d", manga.Chapters),
			Inline: true,
		})
	}

	if manga.Volumes > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Volumes",
			Value:  fmt.Sprintf("%d", manga.Volumes),
			Inline: true,
		})
	}

	if manga.Status != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  manga.Status,
			Inline: true,
		})
	}

	if manga.Score > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Score",
			Value:  fmt.Sprintf("%.2f", manga.Score),
			Inline: true,
		})
	}

	if len(manga.Genres) > 0 {
		genreNames := make([]string, 0, 5)
		for i, g := range manga.Genres {
			if i >= 5 {
				break
			}
			genreNames = append(genreNames, g.Name)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Genres",
			Value:  strings.Join(genreNames, ", "),
			Inline: true,
		})
	}

	if len(manga.Authors) > 0 {
		authorNames := make([]string, 0, 3)
		for i, a := range manga.Authors {
			if i >= 3 {
				break
			}
			authorNames = append(authorNames, a.Name)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Author",
			Value:  strings.Join(authorNames, ", "),
			Inline: true,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Data from MyAnimeList via Jikan API",
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	fmt.Printf("Manga: %s\n", manga.Title)
	return nil
}

// CharacterCommand searches for anime/manga characters
type CharacterCommand struct{}

func (c *CharacterCommand) Name() string        { return "character" }
func (c *CharacterCommand) Aliases() []string   { return []string{"char"} }
func (c *CharacterCommand) Description() string { return "Search for an anime/manga character" }
func (c *CharacterCommand) Usage() string       { return "character <name>" }
func (c *CharacterCommand) RequiredPermissions() []int64 { return nil }
func (c *CharacterCommand) MasterOnly() bool    { return false }

func (c *CharacterCommand) Execute(ctx *Context) error {
	if len(ctx.Args) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Usage: `"+ctx.GetPrefix()+"character <name>`")
		}
		return nil
	}

	query := strings.Join(ctx.Args, " ")
	searchURL := fmt.Sprintf("%s/characters?q=%s&limit=1", jikanBaseURL, url.QueryEscape(query))

	resp, err := httpClient.Get(searchURL)
	if err != nil {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "Error connecting to MyAnimeList API.")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, "API rate limited. Please try again.")
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result JikanCharacterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		if ctx.Message != nil {
			ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("No character found for `%s`", query))
		}
		return nil
	}

	char := result.Data[0]

	embed := &discordgo.MessageEmbed{
		Title: char.Name,
		URL:   char.URL,
		Color: 0xFF003D,
	}

	if char.NameKanji != "" {
		embed.Description = char.NameKanji
	}

	if char.Images.JPG.ImageURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: char.Images.JPG.ImageURL,
		}
	}

	if char.About != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "About",
			Value:  truncate(char.About, 1000),
			Inline: false,
		})
	}

	if char.Favorites > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Favorites",
			Value:  fmt.Sprintf("%d", char.Favorites),
			Inline: true,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Data from MyAnimeList via Jikan API",
	}

	if ctx.Message != nil {
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	fmt.Printf("Character: %s\n", char.Name)
	return nil
}
