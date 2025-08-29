package amway

import (
	"amway/config"
	"amway/model"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// SendSubmissionToReviewChannel sends a submission to the review channel with appropriate formatting.
func SendSubmissionToReviewChannel(s *discordgo.Session, submission *model.Submission, cacheID string) {
	reviewChannelID := config.Cfg.AmwayBot.Amway.ReviewChannelID
	if reviewChannelID == "" {
		log.Printf("Review channel ID not configured")
		return
	}

	var embed *discordgo.MessageEmbed
	// Differentiate between legacy and new submissions based on RecommendTitle
	if submission.RecommendTitle == "" {
		// Legacy submission format
		embed = &discordgo.MessageEmbed{
			Title:       "新的投稿待审核",
			Description: fmt.Sprintf("**投稿ID:** %s\n**作者:** <@%s>\n**标题:** %s\n**URL:** %s\n**内容:**\n%s", submission.ID, submission.UserID, submission.OriginalTitle, submission.URL, submission.RecommendContent),
			Color:       0xFFFF00, // Yellow for pending
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("提交时间 • ID: %s", submission.ID),
			},
		}
	} else {
		// New (V2) submission format
		title := "新的安利投稿待审核"
		if submission.IsAnonymous {
			title += " (匿名)"
		}
		embed = &discordgo.MessageEmbed{
			Title:       title,
			Description: fmt.Sprintf("**投稿ID:** %s\n**投稿人:** <@%s>\n**安利标题:** %s\n**原帖作者:** <@%s>\n**原帖链接:** %s\n**安利内容:**\n%s", submission.ID, submission.UserID, submission.RecommendTitle, submission.OriginalAuthor, submission.URL, submission.RecommendContent),
			Color:       0xFFFF00, // Yellow for pending
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("提交时间 • ID: %s", submission.ID),
			},
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "通过",
					Style:    discordgo.SuccessButton,
					CustomID: "vote:pass:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "不通过",
					Style:    discordgo.DangerButton,
					CustomID: "vote:reject:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
				discordgo.Button{
					Label:    "封禁",
					Style:    discordgo.DangerButton,
					CustomID: "vote:ban:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🔨"},
				},
				discordgo.Button{
					Label:    "精选",
					Style:    discordgo.PrimaryButton,
					CustomID: "vote:feature:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🌟"},
				},
				discordgo.Button{
					Label:    "悔票",
					Style:    discordgo.SecondaryButton,
					CustomID: "vote:remove:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🗑️"},
				},
			},
		},
	}

	_, err := s.ChannelMessageSendComplex(reviewChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
	})

	if err != nil {
		fmt.Printf("Error sending review message: %v\n", err)
	}
}
