package amway_admin

import (
	"amway/db"
	"amway/utils"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleBanUser handles banning a user, either temporarily or permanently.
func handleBanUser(s *discordgo.Session, i *discordgo.InteractionCreate, userID string, durationStr string) {
	if userID == "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr("❌ 请提供需要封禁的用户ID "),
		})
		return
	}

	// If no duration is provided, apply a permanent ban.
	if durationStr == "" {
		err := db.ApplyPermanentBan(userID)
		if err != nil {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("❌ 永久封禁用户 %s 失败: %v", userID, err)),
			})
			return
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("✅ 用户 <@%s> 已被永久封禁", userID)),
		})
		return
	}

	// If a duration is provided, parse it and apply a temporary ban.
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 无效的时长格式: %v. 请使用例如 '72h', '3d' 的格式", err)),
		})
		return
	}

	_, err = db.ApplyBan(userID, duration)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 临时封禁用户 %s 失败: %v", userID, err)),
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.StringPtr(fmt.Sprintf("✅ 用户 <@%s> 已被临时封禁，时长: %s", userID, durationStr)),
	})
}

// handleLiftBan removes a ban from a user.
func handleLiftBan(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	if userID == "" {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr("❌ 请提供需要解除封禁的用户ID "),
		})
		return
	}

	err := db.LiftBan(userID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(fmt.Sprintf("❌ 解除用户 %s 的封禁失败：%v", userID, err)),
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: utils.StringPtr(fmt.Sprintf("✅ 用户 <@%s> 的封禁已解除 ", userID)),
	})
}
