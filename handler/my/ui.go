package my

import (
	"amway/model"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	PageSize = 3 // 每页显示3条投稿
)

// BuildMyAmwayPanelComponents builds the message components for the "My Amway" panel.
// It displays a user profile card followed by a paginated list of submission cards.
func BuildMyAmwayPanelComponents(user *discordgo.User, submissions []*model.Submission, page, totalSubmissions int) (*discordgo.InteractionResponseData, error) {
	var embeds []*discordgo.MessageEmbed

	// 1. Build User Profile Embed (always the first embed)
	profileEmbed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    user.Username,
			IconURL: user.AvatarURL(""),
		},
		Title: "我的安利资料",
		Color: 0x5865F2, // Discord Blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "投稿总数",
				Value:  strconv.Itoa(totalSubmissions),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if totalSubmissions == 0 {
		profileEmbed.Description = "您还没有任何投稿记录。"
	}
	embeds = append(embeds, profileEmbed)

	// 2. Build Submission Embeds for the current page
	for _, submission := range submissions {
		statusEmoji := getStatusEmoji(submission.Status)
		timestamp := time.Unix(submission.Timestamp, 0).Format("2006-01-02 15:04")

		// Truncate content if it exceeds 1024 characters
		content := submission.RecommendContent
		if len(content) > 1024 {
			content = content[:1021] + "..."
		}

		// Add author and original post link
		var extraInfo strings.Builder
		if submission.OriginalAuthor != "" {
			extraInfo.WriteString(fmt.Sprintf("\n\n**作者:** <@%s>", submission.OriginalAuthor))
		}
		if submission.URL != "" {
			extraInfo.WriteString(fmt.Sprintf("\n**原帖链接:** %s", submission.URL))
		}
		if extraInfo.Len() > 0 {
			content += "\n---" + extraInfo.String()
		}

		submissionEmbed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s 安利ID: %s | %s", statusEmoji, submission.ID, submission.RecommendTitle),
			Description: content,
			Color:       0x7D8B99, // A slightly different color for submission cards
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("提交于: %s | 状态: %s", timestamp, submission.Status),
			},
		}
		embeds = append(embeds, submissionEmbed)
	}

	// 3. Pagination Logic & Buttons
	totalPages := (totalSubmissions + PageSize - 1) / PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	prevButton := discordgo.Button{
		Label:    "⬅️ 上一页",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("my_amway_page:%s:%d", user.ID, page-1),
		Disabled: page <= 1,
	}

	nextButton := discordgo.Button{
		Label:    "下一页 ➡️",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("my_amway_page:%s:%d", user.ID, page+1),
		Disabled: page >= totalPages,
	}

	retractButton := discordgo.Button{
		Label:    "🗑️ 撤回投稿",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("retract_submission_button:%s", user.ID),
	}

	// Add a page indicator
	messageContent := fmt.Sprintf("第 %d / %d 页", page, totalPages)
	if totalSubmissions == 0 {
		messageContent = "无投稿记录"
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{prevButton, nextButton, retractButton},
		},
	}

	return &discordgo.InteractionResponseData{
		Content:    messageContent,
		Embeds:     embeds,
		Components: components,
		Flags:      discordgo.MessageFlagsEphemeral,
	}, nil
}

// BuildRetractModal builds the modal for retracting a submission.
func BuildRetractModal(userID string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("retract_submission_modal:%s", userID),
			Title:    "撤回投稿",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_id_to_retract",
							Label:       "请输入要撤回的投稿ID",
							Style:       discordgo.TextInputShort,
							Placeholder: "例如：123",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func getStatusEmoji(status string) string {
	switch status {
	case "approved":
		return "✅"
	case "featured":
		return "🚀"
	case "rejected":
		return "❌"
	case "banned":
		return "🔨"
	case "retracted":
		return "↩️"
	default:
		return "⏳" // Pending or unknown
	}
}

// ParseMyAmwayPageCustomID parses the custom ID for page navigation.
func ParseMyAmwayPageCustomID(customID string) (userID string, page int, err error) {
	parts := strings.Split(customID, ":")
	if len(parts) != 3 {
		return "", 0, fmt.Errorf("invalid customID format")
	}
	p, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid page number in customID")
	}
	return parts[1], p, nil
}
