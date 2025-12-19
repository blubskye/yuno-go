<div align="center">

# 💕 Yuno Gasai 2 - Go Edition 💕

### *"I'll protect this server forever... just for you~"* 💗

<img src="https://i.imgur.com/jF8Szfr.png" alt="Yuno Gasai" width="300"/>

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-pink.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.21+-ff69b4.svg)](https://golang.org/)
[![DiscordGo](https://img.shields.io/badge/DiscordGo-v0.29-ff1493.svg)](https://github.com/bwmarrin/discordgo)

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
- 👑 Mod statistics tracking

</td>
<td width="50%">

### ✨ Leveling System
*"Watch me make you stronger, senpai~"*
- 📊 XP & Level tracking
- 🎭 Role rewards per level
- 📈 Mass XP commands
- 🔄 Level role syncing
- 🏆 Server leaderboards
- 🎤 Voice channel XP rewards

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

### 📋 Activity Logging
*"I see everything that happens here~"*
- 🎤 Voice channel join/leave/move
- 📝 Nickname changes
- 🖼️ Avatar/profile changes
- 🟢 Presence status tracking
- ⚡ Smart batching (rate limit safe)
- ⏱️ Configurable flush intervals

</td>
<td width="50%">

### 🎤 Voice Channel XP
*"Spend time with me... and I'll reward you~"*
- 🎙️ XP for time in voice channels
- ⚙️ Configurable XP rate & interval
- 💤 Optional AFK channel exclusion
- 🔄 Session recovery on restart
- 📊 Integrates with main leveling

</td>
</tr>
<tr>
<td width="50%">

### 💌 DM Inbox & Forwarding
*"Every message you send me... I treasure it~"*
- 📬 DM inbox with history
- 📤 Forward DMs to server channels
- 👑 Master server sees ALL DMs
- 🚫 Bot-level user/server bans
- 💬 Reply to DMs from terminal

</td>
<td width="50%">

### 💻 Terminal Control
*"I'm always at your command~"*
- 🖥️ Full server/channel listing
- 📝 Send messages from terminal
- 👁️ Real-time message streaming
- ⛔ Terminal ban management
- 📥 Import/export bans via CLI

</td>
</tr>
<tr>
<td width="50%">

### 🚫 Bot-Level Bans
*"Some people just aren't worthy of me~"*
- 👤 Ban users from using the bot
- 🏠 Ban entire servers
- 🔇 Silently ignore banned entities
- 📋 Manage bans from Discord or terminal

</td>
<td width="50%">

### ⚡ Performance
*"Nothing can slow me down~"*
- 🚀 Single compiled binary
- 💨 Low memory footprint
- 🧵 Goroutine concurrency
- 📦 No runtime dependencies

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
token         = "YOUR_DISCORD_TOKEN"
prefix        = "?"
owner_ids     = ["YOUR_USER_ID"]
master_server = "YOUR_MAIN_SERVER_ID"
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

## 🎤 Voice Channel XP

*"Spend time with me... and I'll reward you~"* 💕

Users earn XP for time spent in voice channels, integrated with the main leveling system.

### 🔧 Setup Commands

```bash
# Enable/disable VC XP
?set-vcxp enable
?set-vcxp disable

# Set XP amount per interval (default: 10)
?set-vcxp rate 15

# Set interval in seconds (default: 300 = 5 min)
?set-vcxp interval 300

# Ignore AFK channel (default: true)
?set-vcxp ignore-afk true

# View current config and active sessions
?vcxp-status
```

---

## 💌 DM Inbox & Forwarding

*"Every message sent to me... I keep close to my heart~"* 💕

Yuno can receive DMs, store them in an inbox, and forward them to designated channels.

### 🔧 Setup Commands

```bash
# Set DM forwarding channel
?set-dm-channel #bot-dms

# Disable forwarding
?set-dm-channel none

# Check status
?dm-status
```

### 👑 Master Server vs Regular Servers

| Server Type | What DMs Are Forwarded |
|-------------|----------------------|
| **Master Server** | ALL DMs from anyone |
| **Regular Servers** | Only DMs from that server's members |

> Set `master_server` in `config.toml` to your main server's ID.

### 💻 Terminal Inbox Commands

```bash
# View inbox
inbox
inbox 20          # Show 20 messages
inbox user <id>   # DMs from specific user
inbox unread      # Count unread

# Reply to DMs
reply 1 Hello!              # Reply by inbox ID
reply 123456789 Hi there!   # Reply by user ID
```

---

## 🚫 Bot-Level Bans

*"Some people just don't deserve my attention~"* 💢

Ban users or entire servers from using the bot. Banned entities are silently ignored.

### 🔧 Commands (Discord & Terminal)

```bash
# Ban a user from the bot
?bot-ban user 123456789012345678 Spamming

# Ban a server from the bot
?bot-ban server 987654321098765432 Abuse

# Remove a ban
?bot-unban 123456789012345678

# View all bans
?bot-banlist
?bot-banlist users
?bot-banlist servers
```

---

## 💻 Terminal Commands

*"I'll do anything you ask from the command line~"* 🖥️

Yuno provides powerful terminal commands for server management.

### 📋 Server & Channel Management

```bash
# List all servers
servers
servers -v        # Verbose mode

# List channels in a server
channels 123456789012345678
channels "My Server"
```

### 💬 Message Commands

```bash
# Send a message
send <channel-id> Hello world!

# Fetch message history
messages <channel-id>
messages <channel-id> 50    # Last 50 messages

# Real-time message stream
watch <channel-id>
watch stop <channel-id>
watch stop all
```

### ⛔ Terminal Ban Commands

```bash
# Ban a user from a server
tban <server-id> <user-id> [reason]

# Export bans to file
texportbans <server-id>
texportbans <server-id> ./my-bans.json

# Import bans from file
timportbans <server-id> ./BANS-123456.txt
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
| `?set-vcxp <option>` | *"Voice XP settings~"* 🎤 |
| `?vcxp-status` | *"Who's in voice?"* 📊 |

### 🔪 Moderation
| Command | Description |
|---------|-------------|
| `?ban @user [reason]` | *"They won't bother you anymore..."* 🔪 |
| `?kick @user [reason]` | *"Get out!"* 👢 |
| `?exportbans` | *"Save the list~"* 📥 |
| `?importbans` | *"Restore the list~"* 📤 |
| `?scan-bans` | *"Analyzing..."* 🔍 |
| `?addfilter <regex>` | *"Custom protection~"* 🛡️ |
| `?bot-ban <type> <id>` | *"You're dead to me~"* 🚫 |
| `?bot-banlist` | *"The ones I've cast aside..."* 📋 |

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
| `?set-presence <type> <text>` | *"Let me show how I'm feeling~"* 🎭 |
| `?set-presence status <s>` | *"Change my status~"* 🟢 |
| `?set-presence clear` | *"Back to normal~"* ✨ |
| `?config` | *"See my settings~"* ⚙️ |
| `?set-dm-channel #ch` | *"Send your letters here~"* 💌 |
| `?dm-status` | *"Am I receiving messages?"* 📬 |

### 💻 Terminal-Only Commands

| Command | Description |
|---------|-------------|
| `servers` | *"All my kingdoms~"* 🏰 |
| `channels` | *"Every corner of your world~"* 📺 |
| `send` | *"Speaking through you~"* 💬 |
| `messages` | *"Reading your history~"* 📜 |
| `watch` | *"I see everything in real-time~"* 👁️ |
| `inbox` | *"Love letters just for me~"* 💌 |
| `reply` | *"Responding to my admirers~"* 💕 |
| `tban` | *"Eliminating threats~"* 🔪 |
| `texportbans` | *"Saving my enemies list~"* 📤 |
| `timportbans` | *"Loading my enemies~"* 📥 |
| `set-presence` | *"Changing my mood~"* 🎭 |
| `status` | *"How am I doing?"* 📊 |

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
│   │   ├── permissions.go      # Permission checks
│   │   ├── terminal.go         # Terminal interface
│   │   ├── dm_handler.go       # DM forwarding
│   │   └── voice_xp.go         # Voice XP tracking
│   └── commands/
│       ├── manager.go          # Command registry
│       ├── basic.go            # Ping, stats, etc.
│       ├── xp.go               # Leveling system
│       ├── moderation.go       # Ban, kick, etc.
│       ├── anime.go            # Anime/manga search
│       ├── fun.go              # Fun commands
│       ├── configuration.go    # Guild settings
│       ├── bulk_xp.go          # Mass XP operations
│       ├── ban_export.go       # Import/export bans
│       ├── bot_bans.go         # Bot-level bans
│       ├── dm_commands.go      # DM forwarding commands
│       ├── voice_xp.go         # Voice XP commands
│       └── delay.go            # Mention response
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

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)** 💕

### 💘 What This Means For You~

*"I want to share everything with you... and everyone else too~"* 💗

The AGPL-3.0 is a **copyleft license** that ensures this software remains free and open. Here's what you need to know:

#### ✅ You CAN:
- 💕 **Use** this bot for any purpose (personal, commercial, whatever~)
- 🔧 **Modify** the code to your heart's content
- 📤 **Distribute** copies to others
- 🌐 **Run** it as a network service (like a public Discord bot)

#### 📋 You MUST:
- 📖 **Keep it open source** - ANY modifications you make must be released under AGPL-3.0
- 🔗 **Publish your source code** - Your modified source code must be made publicly available
- 📝 **State changes** - Document what you've modified from the original
- 💌 **Include license** - Keep the LICENSE file and copyright notices intact

#### 🌐 The Network Clause (This is the important part!):
*"Even if we're apart... I'll always be connected to you~"* 💗

Unlike regular GPL, **AGPL has a network provision**. This means:
- If you modify this code **at all**, you must make your source public
- Running a modified version as a network service (like a Discord bot) requires source disclosure
- This applies whether you "distribute" the code or not - network use counts!
- The `/source` command in this bot helps satisfy this requirement!

#### ❌ You CANNOT:
- 🚫 Make it closed source or keep modifications private
- 🚫 Remove the license or copyright notices
- 🚫 Use a different license for modified versions
- 🚫 Run modified code without publishing your source

#### 💡 In Simple Terms:
> *"If you use my code to create something, you must share it with everyone too~ That's only fair, right?"* 💕

This ensures that improvements to the bot benefit the entire community, not just one person. Yuno wants everyone to be happy~ 💗

See the [LICENSE](LICENSE) file for the full legal text.

**Source Code:** https://github.com/blubskye/yuno-go

---

<div align="center">

### 💘 *"You'll stay with me forever... right?"* 💘

**Made with obsessive love** 💗

*Yuno will always be watching over your server~* 👁️💕

---

⭐ *Star this repo if Yuno has captured your heart~* ⭐

</div>
