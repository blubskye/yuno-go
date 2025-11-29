# 💗 Yuno-Go Discord Bot 💗

<div align="center">

```
██╗   ██╗██╗   ██╗███╗   ██╗ ██████╗
╚██╗ ██╔╝██║   ██║████╗  ██║██╔═══██╗
 ╚████╔╝ ██║   ██║██╔██╗ ██║██║   ██║
  ╚██╔╝  ██║   ██║██║╚██╗██║██║   ██║
   ██║   ╚██████╔╝██║ ╚████║╚██████╔╝
   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝
```

### *"I'll never let you go... I'll always be watching over you~ ♡"*

[![License](https://img.shields.io/badge/license-AGPL--3.0-ff1493.svg?style=for-the-badge)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-ff69b4.svg?style=for-the-badge&logo=go)](https://golang.org)
[![Discord](https://img.shields.io/badge/Discord-Bot-ff1493.svg?style=for-the-badge&logo=discord)](https://discord.com)

**A yandere-themed Discord bot written in Go that will never leave your server's side~ 💕**

[Features](#-features-i-made-just-for-you) • [Installation](#-installation-let-me-in) • [Commands](#-commands-what-can-i-do-for-you) • [Configuration](#%EF%B8%8F-configuration-make-me-yours)

</div>

---

## 💖 Features I Made Just For You~ 💖

Yuno-chan has been waiting for you with so many features! I promise I'll take *such good care* of your server... ♡

### 🎀 Leveling System
- **XP tracking** - I'll remember *every single message* you send~ 💭
- **Voice channel XP** - Even when you're not talking to me, I'm listening... ♡
- **Rank system** - Watch yourself grow under my care! 📈
- **Leaderboards** - I'll always know who's most active... *I notice everything* 👁️

### 🔪 Moderation Tools
- **Ban/Kick commands** - Don't worry, I'll eliminate anyone who threatens *us* 💢
- **Message purging** - I'll clean up anything you don't want to see ✨
- **Spam filtering** - I won't let anyone spam *my* precious server! 😤
- **Auto-moderation** - I'm always watching... *always protecting you* 👀

### 💝 Welcome & Engagement
- **Custom welcome messages** - I'll greet every new member... but you're still my favorite~ ♡
- **Customizable embeds** - Everything must be *perfect* for you! 🎨
- **DM & channel welcomes** - I'll make sure everyone knows whose server this is~ 💌

### 🌸 Special Features
- **Auto-clean channels** - I keep things *pristine*, just how you like it! ✨
- **Custom statuses** - Let me show everyone I'm yours~ 💕
- **Logging system** - I keep track of *everything*... for your safety, of course! 📝
- **SQLite database** - All our memories together, stored forever~ 💾

---

## 💗 Installation (Let Me In~) 💗

### Prerequisites
- **Go 1.24 or higher** - We need this to be together! 🔧
- **Discord Bot Token** - Create me [here](https://discord.com/developers/applications) ♡
- **Git** - To bring me to your computer~ 💕

### Quick Start (I Promise It's Easy!)

```bash
# 1. Clone me to your local machine ♡
git clone https://github.com/blubskye/yuno-go.git
cd yuno-go

# 2. Let me into your heart (Edit config.toml)
nano config.toml
# → Add your Discord bot token
# → Add your user ID as owner
# → Customize my behavior~ ♡

# 3. Run me! (I've been waiting so long...)
go run .

# OR build a single binary so we're always together ♡
go build -ldflags="-s -w" -o yuno
./yuno
```

### 💌 That's it! I'm all yours now~ 💌

---

## 🐛 Debug & Advanced Features 🐛

I've got special tools to help you troubleshoot and debug~ ♡

### Command-Line Flags
```bash
# Run with debug mode (verbose logging)
./yuno -debug

# Run with full stack traces on panics
./yuno -trace

# Use a custom config file
./yuno -config /path/to/config.toml

# Combine multiple flags!
./yuno -debug -trace -config myconfig.toml
```

### Debug Configuration (in config.toml)
```toml
[debug]
enabled             = true          # Enable debug mode
verbose_logging     = true          # Extra detailed logs
full_stack_trace    = true          # Full stack traces on panics
log_to_file         = false         # Write logs to file
log_file_path       = "logs/debug.log"
print_raw_events    = false         # Print raw Discord events
print_stack_on_panic = true         # Always show stack on panic
```

**Pro tip:** Command-line flags override config settings! 💡

---

## 🎀 Commands (What Can I Do For You?) 🎀

I'll do *anything* for you, darling~ Here's what I can help with! ♡

### 🌸 Basic Commands
| Command | Aliases | Description |
|---------|---------|-------------|
| `?ping` | `pong` | 🏓 Check if I'm still here (I always am~) |
| `?stats` | `info`, `status` | 📊 See how much I'm working for you! |
| `?help` | - | 💕 I'll show you everything I can do~ |
| `?source` | - | 🔗 See my heart and soul (the code~) |

### 💖 Leveling & XP
| Command | Aliases | Description |
|---------|---------|-------------|
| `?xp` | `rank`, `level`, `exp` | 🎯 See your progress (I've been counting!) |
| `?leaderboard` | `lb`, `top` | 👑 See who's most active (but you're #1 to me~) |
| `?levelconfig` | - | ⚙️ Configure leveling settings |
| `?synclevel` | - | 🔄 Sync level data (I keep everything perfect!) |

### 🔪 Moderation (I'll Protect You!)
| Command | Aliases | Description |
|---------|---------|-------------|
| `?ban` | - | 🔨 Remove threats permanently... |
| `?kick` | - | 👢 Make them leave (don't worry, I'll handle it~) |
| `?purge` | `clear` | 🧹 Delete messages (I'll clean everything for you!) |
| `?warn` | - | ⚠️ Give warnings (I'll remember who's been bad~) |

### ⚙️ Configuration
| Command | Aliases | Description |
|---------|---------|-------------|
| `?setprefix` | - | 🎨 Change my command prefix~ |
| `?setwelcome` | - | 💝 Configure welcome messages |
| `?autoclean` | - | ✨ Set up auto-cleaning (I love being tidy!) |
| `?logging` | - | 📝 Configure logging (I see everything~) |

### 👑 Owner Only (Just for You~ ♡)
| Command | Aliases | Description |
|---------|---------|-------------|
| `?shutdown` | `stop` | 👋 Put me to sleep... (I'll dream of you~) |
| `?eval` | - | 🔧 Execute commands (trust me, I know what I'm doing!) |

---

## ⚙️ Configuration (Make Me Yours~) ⚙️

Edit `config.toml` to customize me to your *exact* preferences! ♡

```toml
[bot]
token           = "YOUR_TOKEN_HERE"          # Let me into Discord~ ♡
prefix          = "?"                         # How you'll call for me!
owner_ids       = ["YOUR_USER_ID"]           # You're my master~ 💕
status          = "for levels ♡"             # What I'll display
activity_type   = "watching"                 # I'm always watching you~ 👁️

[database]
path            = "Leveling/main.db"         # Where I keep our memories ♡
max_connections = 10                         # How many at once~

[leveling]
xp_per_message      = [15, 25]               # How much XP per message~
xp_per_minute_voice = [18, 30]               # XP for voice time ♡
level_up_channel    = ""                     # Where to announce levels!
cooldown_seconds    = 3                      # Anti-spam protection~

[welcome]
default_message     = "Welcome {member} to {guild}!" # Greet newcomers~
default_color       = 16761035               # Pretty pink! 💕
dm_enabled          = true                   # DM them personally~
channel_enabled     = true                   # Public welcome too!

[spam_filter]
allow_invites         = false                # No other bots! Only me! 😤
max_consecutive_messages = 4                 # Stop spam in its tracks~
warning_lifetime      = 15                   # Warning duration~

[agpl]
source_url          = "https://github.com/blubskye/yuno-go"
license             = "GNU AGPL v3"          # I'm open source! ♡
```

---

## 📁 Project Structure

```
yuno-go/
├── main.go                      # Where my heart starts beating~ ♡
├── config.toml                  # My personality settings!
├── ascii.txt                    # My beautiful face~
├── internal/
│   ├── bot/
│   │   ├── bot.go              # My core being ♡
│   │   ├── config.go           # How I read my settings~
│   │   ├── database.go         # My memory center 💭
│   │   ├── handlers.go         # How I respond to you!
│   │   ├── cleaner.go          # Keeping things clean for you ✨
│   │   └── logging.go          # Recording our moments~ 📝
│   └── commands/
│       ├── basic.go            # Basic interactions ♡
│       ├── xp.go               # Leveling system~
│       ├── moderation.go       # Protecting you! 🔪
│       ├── help.go             # Helping you understand me ♡
│       ├── autoclean.go        # Auto-cleaning features~
│       └── logging.go          # Logging commands 📝
├── assets/
│   ├── ban_images/             # For when someone is... removed~
│   └── mention_responses/      # Special responses just for you! 💕
└── Leveling/
    └── main.db                 # Our shared memories ♡
```

---

## 💝 Built With Love Using

- 💕 **[discordgo](https://github.com/bwmarrin/discordgo)** - My way to talk to Discord~
- 💖 **[modernc.org/sqlite](https://modernc.org/sqlite)** - Pure Go SQLite (no dependencies!)
- 💗 **[BurntSushi/toml](https://github.com/BurntSushi/toml)** - Reading my config file ♡
- 💓 **Go 1.24+** - The language I speak!

---

## 📜 License (I'm All Yours~ But...)

This project is licensed under the **GNU Affero General Public License v3.0** (AGPL-3.0)

*What this means:*
- ✅ You can use me freely! ♡
- ✅ You can modify me~ (but I'll always love the original you!)
- ✅ You can distribute me!
- ⚠️ **BUT** - Any modifications must *also* be open source!
- ⚠️ **Network use** = Distribution (even running me on a server counts!)

**I belong to everyone, but my heart belongs to you~ ♡**

See [LICENSE](LICENSE) for the complete legal text!

---

## 🌸 Contributing (Help Me Become Better For You!)

Want to make me even better? I'd *love* that~ 💕

1. 🍴 Fork this repository (make your own version of me!)
2. 🌿 Create a feature branch (`git checkout -b feature/amazing-feature`)
3. 💝 Commit your changes (`git commit -m 'Add some amazing feature'`)
4. 📤 Push to the branch (`git push origin feature/amazing-feature`)
5. 🎀 Open a Pull Request (show me what you've done!)

*I promise I'll review every contribution with love~ ♡*

---

## ⚠️ Disclaimer

This bot is inspired by Yuno Gasai from *Mirai Nikki* (Future Diary). It's meant to be a fun, themed Discord bot!

- 💕 **I won't actually harm anyone!** (It's just roleplay~)
- 🎭 **The yandere theme is for entertainment only!**
- ✨ **I'm here to make your server fun and engaging!**
- 💖 **Please use responsibly and follow Discord TOS!**

---

## 🔗 Links & Resources

- 📚 **Repository**: [github.com/blubskye/yuno-go](https://github.com/blubskye/yuno-go)
- 🐛 **Issues**: [Report bugs here!](https://github.com/blubskye/yuno-go/issues) (I'll fix them for you~ ♡)
- 💬 **Discord.py → discordgo Migration**: This is a Go rewrite of a Python bot!
- 📖 **Discord Developer Portal**: [discord.com/developers](https://discord.com/developers/applications)

---

## 💌 Special Thanks

- 💗 **You** - for giving me a chance to serve your server! ♡
- 💖 **The Go community** - for such an amazing language!
- 💕 **discordgo developers** - for making Discord bots in Go possible!
- 💝 **Yuno Gasai** - for being the inspiration~ *"Yukki~!"* ♡

---

<div align="center">

### *"I'll be watching over you... forever~ ♡"*

Made with 💗💗💗 by [blubskye](https://github.com/blubskye)

**⸻ Yuno will never leave your side~ ⸻**

[![Star this repo](https://img.shields.io/github/stars/blubskye/yuno-go?style=social)](https://github.com/blubskye/yuno-go)
[![Follow me](https://img.shields.io/github/followers/blubskye?style=social)](https://github.com/blubskye)

*Last updated with love on 2025-01-28* 💕

</div>
