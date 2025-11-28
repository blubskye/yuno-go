package bot

import (
	"log"
	"math"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// giveXP handles XP calculation and level ups
func (b *Bot) giveXP(s *discordgo.Session, guildID, userID, channelID string, xp int) {
	tx, err := b.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	var currentXP, level int
	var enabled string
	err = tx.QueryRow(`
		SELECT exp, level, enabled FROM glevel 
		WHERE guild_id = ? AND user_id = ?`, guildID, userID).
		Scan(&currentXP, &level, &enabled)

	if err != nil {
		// First time seeing this user
		_, err = tx.Exec(`INSERT INTO glevel (guild_id, user_id, exp, level, enabled) VALUES (?, ?, ?, ?, 'enabled')`,
			guildID, userID, 0, 0)
		if err != nil {
			log.Printf("Failed to insert new user: %v", err)
			return
		}
		currentXP = 0
		level = 0
		enabled = "enabled"
	}

	if enabled != "enabled" {
		return
	}

	newXP := currentXP + xp
	newLevel := int(math.Floor((math.Sqrt(1 + 8*float64(newXP)/50) - 1) / 2))

	if newLevel > level {
		user, err := s.User(userID)
		if err == nil {
			s.ChannelMessageSend(channelID,
				user.Mention()+" just reached **Level "+strconv.Itoa(newLevel)+"**! ♡")
		}
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO glevel (guild_id, user_id, exp, level, enabled)
	         VALUES (?, ?, ?, ?, 'enabled')`, guildID, userID, newXP, newLevel)
	if err != nil {
		log.Printf("Failed to update XP: %v", err)
		return
	}
	
	tx.Commit()

	// Auto-role logic (you'll expand this later)
	b.checkLevelRoles(guildID, userID, newLevel)
}

func (b *Bot) checkLevelRoles(guildID, userID string, level int) {
	rows, err := b.DB.Query(`
		SELECT role_id FROM ranks WHERE guild_id = ? AND level <= ?`, guildID, level)
	if err != nil {
		return
	}
	defer rows.Close()

	var roleID string
	for rows.Next() {
		rows.Scan(&roleID)
		go b.Session.GuildMemberRoleAdd(guildID, userID, roleID)
	}
}

func (b *Bot) onMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	// Welcome system placeholder – fully ported next if you want
	log.Printf("%s joined %s", m.User.String(), m.GuildID)
}

// Voice XP stub – ready for full implementation
func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Coming in Phase 2
}
