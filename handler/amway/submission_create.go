package amway

import (
	"amway/db"
	"amway/model"
	"amway/utils"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func validateUserBanStatus(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	banned, _, err := db.CheckUserBanStatus(i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error checking if user is banned: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法处理您的请求，请稍后再试",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	}

	if banned {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您已被禁止投稿",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	}
	return true
}

func validateSubmissionRateLimit(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	canSubmit, remainingTime := utils.CheckSubmissionRateLimit(i.Member.User.ID)
	if !canSubmit {
		hours := int(remainingTime.Hours())
		minutes := int(remainingTime.Minutes()) % 60

		var timeMsg string
		if hours > 0 {
			timeMsg = fmt.Sprintf("%d小时%d分钟", hours, minutes)
		} else {
			timeMsg = fmt.Sprintf("%d分钟", minutes)
		}

		embed := &discordgo.MessageEmbed{
			Title:       "投稿频率限制",
			Color:       0xFF6B6B, // 红色
			Description: "为了保证安利墙的质量，每位用户3小时内只能投稿一次",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "剩余等待时间",
					Value:  timeMsg,
					Inline: true,
				},
				{
					Name:   "投稿限制",
					Value:  "3小时/次",
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "请耐心等待冷却时间结束",
			},
			Timestamp: fmt.Sprintf("%v", time.Now().Format(time.RFC3339)),
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	}
	return true
}

func parseAndValidateCustomID(s *discordgo.Session, i *discordgo.InteractionCreate, customID string, minParts int) ([]string, bool) {
	parts := strings.Split(customID, ":")
	if len(parts) < minParts {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return nil, false
	}
	return parts, true
}

func validateCacheData(s *discordgo.Session, i *discordgo.InteractionCreate, cacheID string) (model.SubmissionData, bool) {
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您的投稿请求已过期，请重新发起",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return model.SubmissionData{}, false
	}
	return cacheData, true
}

func CreateSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !validateUserBanStatus(s, i) {
		return
	}

	if !validateSubmissionRateLimit(s, i) {
		return
	}

	err := s.InteractionRespond(i.Interaction, BuildSubmissionLinkModal())
	if err != nil {
		fmt.Printf("Error creating modal: %v\n", err)
	}
}

func LinkSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	var url string
	for _, component := range data.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok {
					if textInput.CustomID == "submission_url" {
						url = textInput.Value
						break
					}
				}
			}
		}
	}

	if url == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "请提供有效的链接",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	currentGuildID := i.GuildID
	postInfo, err := utils.ValidateDiscordPost(s, url, currentGuildID, i.Member.User.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("验证失败：%v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embeds, components := BuildPostConfirmationComponents(postInfo)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func ConfirmPostHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 4)
	if !ok {
		return
	}

	channelID := parts[1]
	messageID := parts[2]
	originalAuthor := parts[3]

	cacheData := model.SubmissionData{
		ChannelID:      channelID,
		MessageID:      messageID,
		OriginalAuthor: originalAuthor,
	}
	cacheID := utils.AddToCache(cacheData)

	embeds, components := BuildReplyChoiceComponents(cacheID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
			Embeds:     embeds,
		},
	})

	if err != nil {
		fmt.Printf("Error updating message for reply choice: %v\n", err)
	}
}

func ReplyChoiceHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 3)
	if !ok {
		return
	}

	cacheID := parts[1]
	replyToOriginalStr := parts[2]
	replyToOriginal := replyToOriginalStr == "true"

	cacheData, ok := validateCacheData(s, i, cacheID)
	if !ok {
		return
	}

	cacheData.ReplyToOriginal = replyToOriginal
	utils.UpdateCache(cacheID, cacheData)

	err := s.InteractionRespond(i.Interaction, BuildSubmissionContentModal(cacheID, "", ""))
	if err != nil {
		fmt.Printf("Error creating content modal after reply choice: %v\n", err)
	}
}

func EditSubmissionLinkHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, BuildSubmissionLinkModal())
	if err != nil {
		fmt.Printf("Error creating modal for editing submission link: %v\n", err)
	}
}

func CancelSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: BuildCancelResponseData(),
	})
}

func ContentSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	customID := data.CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 2)
	if !ok {
		return
	}

	cacheID := parts[1]
	cacheData, ok := validateCacheData(s, i, cacheID)
	if !ok {
		return
	}

	var recommendTitle, recommendContent string
	for _, component := range data.Components {
		if actionRow, ok := component.(*discordgo.ActionsRow); ok {
			for _, comp := range actionRow.Components {
				if textInput, ok := comp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "recommend_title":
						recommendTitle = textInput.Value
					case "recommend_content":
						recommendContent = textInput.Value
					}
				}
			}
		}
	}

	if recommendTitle == "" || recommendContent == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "标题和内容都是必填的，请重新提交",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheData.RecommendTitle = strings.TrimLeft(recommendTitle, "#")
	cacheData.RecommendContent = recommendContent
	utils.UpdateCache(cacheID, cacheData)

	embeds, components := BuildSubmissionPreviewComponents(recommendTitle, recommendContent, cacheID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "",
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error updating message with preview: %v\n", err)
	}
}

func FinalSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !validateUserBanStatus(s, i) {
		return
	}

	if !validateSubmissionRateLimit(s, i) {
		return
	}

	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 3)
	if !ok {
		return
	}

	cacheID := parts[1]
	isAnonymousStr := parts[2]
	isAnonymous := isAnonymousStr == "true"

	cacheData, ok := validateCacheData(s, i, cacheID)
	if !ok {
		return
	}
	// Don't remove from cache yet - we need it for voting

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: BuildFinalSuccessResponseData(),
	})
	if err != nil {
		fmt.Printf("Error updating final submission message: %v\n", err)
	}

	guildID := i.GuildID
	originalURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, cacheData.ChannelID, cacheData.MessageID)

	postInfo, err := utils.ValidateDiscordPost(s, originalURL, guildID, i.Member.User.ID)
	var originalPostTimestamp, originalTitle string
	if err == nil && postInfo != nil {
		originalPostTimestamp = postInfo.Timestamp
		originalTitle = postInfo.Title
	} else {
		fmt.Printf("二次验证帖子以获取元数据时出错: %v\n", err)
	}

	submissionID, err := db.AddSubmissionV2(
		i.Member.User.ID, originalURL,
		cacheData.RecommendTitle, cacheData.RecommendContent,
		originalTitle, cacheData.OriginalAuthor, originalPostTimestamp, guildID, i.Member.User.Username, isAnonymous,
	)
	if err != nil {
		fmt.Printf("Error adding submission to database: %v\n", err)
		errorContent := fmt.Sprintf("提交失败，请稍后再试错误详情: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorContent})
		return
	}

	// Record submission time for rate limiting
	utils.RecordSubmissionTime(i.Member.User.ID)

	// Update cache with submission ID for auto-rejection functionality
	cacheData.SubmissionID = submissionID
	utils.UpdateCache(cacheID, cacheData)

	submission := &model.Submission{
		ID:               submissionID,
		UserID:           i.Member.User.ID,
		URL:              originalURL,
		RecommendTitle:   cacheData.RecommendTitle,
		RecommendContent: cacheData.RecommendContent,
		OriginalAuthor:   cacheData.OriginalAuthor,
		IsAnonymous:      isAnonymous,
	}
	SendSubmissionToReviewChannel(s, submission, cacheID)
}

func EditSubmissionContentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 2)
	if !ok {
		return
	}

	cacheID := parts[1]
	cacheData, ok := validateCacheData(s, i, cacheID)
	if !ok {
		return
	}

	err := s.InteractionRespond(i.Interaction, BuildSubmissionContentModal(cacheID, cacheData.RecommendTitle, cacheData.RecommendContent))
	if err != nil {
		fmt.Printf("Error creating modal for editing submission content: %v\n", err)
	}
}

func HowToSubmitButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: BuildHowToSubmitResponseData(),
	})

	if err != nil {
		fmt.Printf("Error sending how-to-submit embed: %v\n", err)
	}
}

func ConfirmPreviewHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 2)
	if !ok {
		return
	}

	cacheID := parts[1]
	_, ok = validateCacheData(s, i, cacheID)
	if !ok {
		return
	}

	embeds, components := BuildAnonymityChoiceComponents(cacheID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error updating message for anonymity choice: %v\n", err)
	}
}

func BackToPreviewHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts, ok := parseAndValidateCustomID(s, i, customID, 2)
	if !ok {
		return
	}

	cacheID := parts[1]
	cacheData, ok := validateCacheData(s, i, cacheID)
	if !ok {
		return
	}

	embeds, components := BuildSubmissionPreviewComponents(cacheData.RecommendTitle, cacheData.RecommendContent, cacheID)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error updating message back to preview: %v\n", err)
	}
}
