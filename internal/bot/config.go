// internal/bot/config.go
package bot

import (
	"github.com/BurntSushi/toml"
	"log"
	"os"
)

type Config struct {
	Bot         BotConfig
	Database    DatabaseConfig
	Features    FeaturesConfig
	Leveling    LevelingConfig
	Welcome     WelcomeConfig
	SpamFilter  SpamFilterConfig
	Terminal    TerminalConfig
	Paths       PathsConfig
	Performance PerformanceConfig
	AGPL        AGPLConfig
}

type BotConfig struct {
	Token        string   `toml:"token"`
	Prefix       string   `toml:"prefix"`
	OwnerIDs     []string `toml:"owner_ids"`
	Status       string   `toml:"status"`
	ActivityType string   `toml:"activity_type"`
}

type DatabaseConfig struct {
	Path            string `toml:"path"`
	BackupInterval  string `toml:"backup_interval"`
	MaxConnections  int    `toml:"max_connections"`
}

type FeaturesConfig struct {
	LevelingEnabledByDefault bool `toml:"leveling_enabled_by_default"`
	WelcomeEnabledByDefault  bool `toml:"welcome_enabled_by_default"`
	SpamFilterEnabled        bool `toml:"spam_filter_enabled"`
	AutoCleanEnabled         bool `toml:"auto_clean_enabled"`
}

type LevelingConfig struct {
	XpPerMessage      []int  `toml:"xp_per_message"`
	XpPerMinuteVoice  []int  `toml:"xp_per_minute_voice"`
	LevelUpChannel    string `toml:"level_up_channel"`
	CooldownSeconds   int    `toml:"cooldown_seconds"`
}

type WelcomeConfig struct {
	DefaultMessage  string `toml:"default_message"`
	DefaultColor    int    `toml:"default_color"`
	DMEnabled       bool   `toml:"dm_enabled"`
	ChannelEnabled  bool   `toml:"channel_enabled"`
	EmbedImageURL   string `toml:"embed_image_url"`
}

type SpamFilterConfig struct {
	MainChannelPrefix     string `toml:"main_channel_prefix"`
	NSFWChannelPrefix     string `toml:"nsfw_channel_prefix"`
	AllowInvites          bool   `toml:"allow_invites"`
	MaxConsecutiveMessages int   `toml:"max_consecutive_messages"`
	WarningLifetime       int    `toml:"warning_lifetime"`
}

type TerminalConfig struct {
	AllowedUsers []string `toml:"allowed_users"`
	LogChannel   string   `toml:"log_channel"`
}

type PathsConfig struct {
	BanImagesFolder     string `toml:"ban_images_folder"`
	MentionImagesFolder string `toml:"mention_images_folder"`
	LogsFolder          string `toml:"logs_folder"`
}

type PerformanceConfig struct {
	GoroutineLimit  int    `toml:"goroutine_limit"`
	CacheTTL        string `toml:"cache_ttl"`
	RateLimitBurst  int    `toml:"rate_limit_burst"`
}

type AGPLConfig struct {
	SourceURL string `toml:"source_url"`
	RepoName  string `toml:"repo_name"`
	License   string `toml:"license"`
}

var Global Config

func LoadConfig(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("config.toml not found → generating default")
		generateDefaultConfig(path)
	}

	if _, err := toml.DecodeFile(path, &Global); err != nil {
		log.Fatal("Failed to load config.toml:", err)
	}

	if Global.Bot.Token == "" || Global.Bot.Token == "YOUR_DISCORD_BOT_TOKEN_HERE" {
		log.Fatal("You must set a valid token in config.toml")
	}

	log.Println("Configuration loaded ♡")
}

func generateDefaultConfig(path string) {
	file, _ := os.Create(path)
	defer file.Close()
	encoder := toml.NewEncoder(file)
	encoder.Encode(Config{
		Bot: BotConfig{
			Prefix:       "?",
			Status:       "for levels ♡",
			ActivityType: "watching",
			OwnerIDs:     []string{"0"}, // placeholder
		},
		// ... other defaults matching the template above
	})
	log.Println("Default config.toml generated — edit it and restart")
}
