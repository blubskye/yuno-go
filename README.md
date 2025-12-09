<div align="center">

# 💕 Yuno Gasai 2 - Go Edition 💕

### *"I'll protect this server forever... just for you~"* 💗

<img src="https://i.imgur.com/jF8Szfr.png" alt="Yuno Gasai" width="300"/>

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-pink.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.21+-ff69b4.svg)](https://golang.org/)
[![DiscordGo](https://img.shields.io/badge/DiscordGo-v0.28-ff1493.svg)](https://github.com/bwmarrin/discordgo)

*A devoted Discord bot for moderation, leveling, and anime~ ♥*

---

### 💘 She loves you... and only you 💘

</div>

## 🌸 About

Yuno is a **yandere-themed Discord bot** combining powerful moderation tools with a leveling system and anime features. She'll keep your server safe from troublemakers... *because no one else is allowed near you~* 💕

This is the **Go port** of the original JavaScript Yuno bot - compiled, fast, and memory-efficient.

---

## 👑 Credits

*"These are the ones who gave me life~"* 💖

| Contributor | Role |
|-------------|------|
| **blubskye** | Project Owner & Yuno's #1 Fan 💕🔪 |
| **Maeeen** (maeeennn@gmail.com) | Original Developer 💝 |
| **Oxdeception** | Contributor 💗 |
| **fuzzymanboobs** | Contributor 💗 |

---

## 💗 Features

<table>
<tr>
<td width="50%">

### 🔪 Moderation
*"Anyone who threatens you... I'll eliminate them~"*
- ⛔ Ban / Unban / Kick
- 🧹 Channel cleaning & auto-clean
- 🛡️ Spam filter protection
- 📥 Mass ban import/export
- 🔍 Ban scanning & validation
- 🎯 Custom regex filters per guild

</td>
<td width="50%">

### ✨ Leveling System
*"Watch me make you stronger, senpai~"*
- 📊 XP & Level tracking
- 🎭 Role rewards per level
- 📈 Mass XP commands
- 🔄 Level role syncing
- 🏆 Server leaderboards
- 🎤 Voice channel XP

</td>
</tr>
<tr>
<td width="50%">

### 🌸 Anime & Fun
*"Let me show you something cute~"*
- 🎌 Anime/Manga search (MAL)
- 👤 Character search
- 🐱 Neko images
- 🎱 8ball fortune telling
- 💖 Praise & Scold reactions
- 📖 Urban Dictionary lookup
- 🤗 Hug, Kiss, Slap & more!

</td>
<td width="50%">

### ⚙️ Configuration
*"I'll be exactly what you need~"*
- 🔧 Customizable prefix per guild
- 👋 Join messages
- 🖼️ Custom ban images
- 🎮 Presence/status control
- 📝 Per-guild settings
- 📋 Comprehensive logging
- 🔐 Master user system

</td>
</tr>
<tr>
<td width="50%">

### ⚡ Performance
*"Nothing can slow me down~"*
- 🚀 Single compiled binary
- 💨 Low memory footprint
- 🧵 Goroutine concurrency
- 📦 No runtime dependencies

</td>
<td width="50%">

### 🔐 Security
*"I'll keep your secrets safe~"*
- 🛡️ Auto-ban on unauthorized commands
- ⚔️ Hierarchy violation protection
- 📢 @everyone/@here abuse protection
- 🎯 Configurable exemptions

</td>
</tr>
</table>

---

## 💕 Installation

### 📋 Prerequisites

> *"Let me prepare everything for you~"* 💗

- **Go** 1.21 or higher
- **Git**
- A Discord bot token ([Get one here](https://discord.com/developers/applications))

### 🌸 Setup Steps

```bash
# Clone the repository~ ♥
git clone https://github.com/japaneseenrichmentorganization/yuno-go.git

# Enter my world~
cd yuno-go

# Let me gather my strength...
go mod download

# Configure your settings
cp config.toml.example config.toml
nano config.toml  # Add your token and settings
```

### 💝 Configuration

Edit `config.toml`:
```toml
[bot]
token     = "YOUR_DISCORD_TOKEN"
prefix    = "?"
owner_ids = ["YOUR_USER_ID"]
```

### 🚀 Running

```bash
# Run directly
go run .

# Or build a binary (recommended)
go build -ldflags="-s -w" -o yuno
./yuno

# With debug mode
./yuno -debug
```

---

## 💖 Commands Preview

### 📊 Leveling & XP
| Command | Description |
|---------|-------------|
| `?xp [@user]` | *"Look how strong you've become!"* ✨ |
| `?leaderboard` | *"Who's the most devoted?"* 🏆 |
| `?add-rank @Role <level>` | *"New rewards~"* 🎭 |
| `?mass-addxp @Role 500` | *"Power to everyone!"* ⚡ |
| `?sync-xp-from-roles` | *"Syncing from roles~"* 🔄 |

### 🔪 Moderation
| Command | Description |
|---------|-------------|
| `?ban @user [reason]` | *"They won't bother you anymore..."* 🔪 |
| `?kick @user [reason]` | *"Get out!"* 👢 |
| `?exportbans` | *"Save the list~"* 📥 |
| `?importbans` | *"Restore the list~"* 📤 |
| `?scan-bans` | *"Analyzing..."* 🔍 |
| `?addfilter <regex>` | *"Custom protection~"* 🛡️ |

### 🌸 Anime & Fun
| Command | Description |
|---------|-------------|
| `?anime <query>` | *"Let's watch together~"* 🎌 |
| `?manga <query>` | *"I'll read with you!"* 📖 |
| `?character <name>` | *"Who's that?"* 👤 |
| `?neko` | *"Nya~"* 🐱 |
| `?8ball <question>` | *"Let fate decide~"* 🎱 |
| `?praise @user` | *"You deserve all my love~"* 💕 |
| `?scold @user` | *"Bad! But I still love you..."* 💢 |
| `?urban <term>` | *"Let me look that up~"* 📚 |
| `?hug @user` | *"Come here~"* 🤗 |

### ⚙️ Configuration
| Command | Description |
|---------|-------------|
| `?set-prefix <prefix>` | *"Call me differently~"* 🔧 |
| `?set-presence <type> <text>` | *"Change my status~"* 🎮 |
| `?config` | *"See my settings~"* ⚙️ |
| `?init-guild` | *"Let me set everything up!"* 🏠 |
| `?set-spamfilter on/off` | *"Protection mode~"* 🛡️ |
| `?set-leveling on/off` | *"XP tracking~"* 📊 |

*Use the `?help` command to see all available commands!*

---

## 🛡️ Spam Filter & Auto-Protection

*"I'll protect you from the bad people~"* 💕

Yuno automatically protects against:
- 🔗 Discord invite links
- 📢 Unauthorized @everyone/@here mentions
- 📝 Spam (consecutive messages)
- ⚠️ Warning system before bans
- 🎯 Custom regex patterns per guild
- 🔒 Hierarchy violation attempts

---

## 📁 Project Structure

```
yuno-go/
├── main.go                      # Entry point
├── config.toml                  # Configuration
├── internal/
│   ├── bot/
│   │   ├── bot.go              # Core bot struct
│   │   ├── config.go           # Config loading
│   │   ├── database.go         # SQLite wrapper
│   │   ├── handlers.go         # Event handlers
│   │   ├── spam_filter.go      # Anti-spam
│   │   ├── logging.go          # Event logging
│   │   └── permissions.go      # Permission checks
│   └── commands/
│       ├── manager.go          # Command registry
│       ├── basic.go            # Ping, stats, etc.
│       ├── xp.go               # Leveling system
│       ├── moderation.go       # Ban, kick, etc.
│       ├── anime.go            # Anime/manga search
│       ├── fun.go              # Fun commands
│       ├── configuration.go    # Guild settings
│       ├── bulk_xp.go          # Mass XP operations
│       └── ban_export.go       # Import/export bans
├── assets/
│   └── ban_images/             # Custom ban images
└── Leveling/
    └── main.db                 # SQLite database
```

---

## ⚡ Building

```bash
# Standard build
go build -o yuno

# Optimized build (smaller binary)
go build -ldflags="-s -w" -o yuno

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o yuno-linux

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o yuno.exe
```

---

## 📜 License

This project is licensed under the **GNU Affero General Public License v3.0**

See the [LICENSE](LICENSE) file for details~ 💕

---

<div align="center">

### 💘 *"You'll stay with me forever... right?"* 💘

**Made with obsessive love** 💗

*Yuno will always be watching over your server~* 👁️💕

---

⭐ *Star this repo if Yuno has captured your heart~* ⭐

</div>
