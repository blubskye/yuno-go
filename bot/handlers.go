package bot

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.GuildID == "" {
		return
	}

	// Simple cooldown: 1 message per user per 3 seconds for XP
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) // anti-raid jitter

	go b.giveXP(m.GuildID, m.Author.ID, rand.Intn(11)+15) // 15–25 XP
}

func (b *Bot) giveXP(guildID, userID string, xp int) {
	tx, _ := b.DB.Begin()
	defer tx.Rollback()

	var currentXP, level int
	var enabled string
	err := tx.QueryRow(`
		SELECT exp, level, enabled FROM glevel 
		WHERE guild_id = ? AND user_id = ?`, guildID, userID).
		Scan(&currentXP, &level, &enabled)

	if err != nil {
		// First time seeing this user
		tx.Exec(`INSERT INTO glevel (guild_id, user_id, exp, level, enabled) VALUES (?, ?, ?, ?, 'enabled')`,
			guildID, userID, 0, 0)
	}

	if enabled != "enabled" {
		return
	}

	newXP := currentXP + xp
	newLevel := int(math.Floor((math.Sqrt(1 + 8*float64(newXP)/50) - 1) / 2))

	if newLevel > level {
		user, _ := s.User(userID)
		s.ChannelMessageSend(m.ChannelID, // you can make this configurable later
			user.Mention()+" just reached **Level "+itoa(newLevel)+"**! ♡")
	}

	tx.Exec(`INSERT OR REPLACE INTO glevel (guild_id, user_id, exp, level, enabled)
	         VALUES (?, ?, ?, ?, 'enabled')`, guildID, userID, newXP, newLevel)
	tx.Commit()

	// Auto-role logic (you’ll expand this later)
	b.checkLevelRoles(guildID, userID, newLevel)
}

func (b *Bot) checkLevelRoles(guildID, userID string, level int) {
	rows, _ := b.DB.Query(`
		SELECT role_id FROM ranks WHERE guild_id = ? AND level <= ?`, guildID, level)
	defer rows.Close()

	var roleID string
	for rows.Next() {
		rows.Scan(&roleID)
		go b.Session.GuildMemberRoleAdd(guildID, userID, roleID)
	}
}

func (b *Bot) onMemberJoin(s *discordgo.Session, m *discordgo.MemberAdd) {
	// Welcome system placeholder — fully ported next if you want
	log.Printf("%s joined %s", m.User.String(), m.GuildID)
}

// Voice XP stub — ready for full implementation
func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Coming in Phase 2
}

// Tiny helper
func itoa(i int) string {
	return string(rune('0'+i)) // only for small numbers, replace later
}
