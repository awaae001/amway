package amway

import (
	"amway/model"
	"amway/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func CreateSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	banned, err := utils.IsUserBanned(i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error checking if user is banned: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法处理您的请求，请稍后再试。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if banned {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您已被禁止投稿",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "submission_link_modal",
			Title:    "投稿第一步：帖子链接",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_url",
							Label:       "Discord帖子链接",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入Discord帖子或频道链接",
							Required:    true,
						},
					},
				},
			},
		},
	})

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
				Content: "请提供有效的链接。",
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

	postInfoText := utils.FormatDiscordPostInfo(postInfo)
	embed := &discordgo.MessageEmbed{
		Title:       "帖子信息确认",
		Description: fmt.Sprintf("%s\n\n请确认以上信息无误，然后点击下方按钮继续填写安利内容。", postInfoText),
		Color:       0x00FF00,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "确认并继续",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_post:%s:%s:%s", postInfo.ChannelID, postInfo.MessageID, postInfo.Author.ID),
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "取消",
					Style:    discordgo.DangerButton,
					CustomID: "cancel_submission",
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func ConfirmPostHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	channelID := parts[1]
	messageID := parts[2]
	originalAuthor := parts[3]

	cacheData := model.SubmissionData{
		ChannelID:      channelID,
		MessageID:      messageID,
		OriginalAuthor: originalAuthor,
		EphChannelID:   i.Message.ChannelID,
		EphMessageID:   i.Message.ID,
	}
	cacheID := utils.AddToCache(cacheData)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("submission_content_modal:%s", cacheID),
			Title:    "投稿第二步：安利内容",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_title",
							Label:       "安利标题",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入您的安利标题",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_content",
							Label:       "安利内容",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "请输入您的安利内容和推荐理由",
							Required:    true,
							MinLength:   20,
							MaxLength:   1024,
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Error creating content modal: %v\n", err)
	}
}

func CancelSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "投稿已取消。",
			Components: []discordgo.MessageComponent{},
			Embeds:     []*discordgo.MessageEmbed{},
		},
	})
}

func ContentSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	customID := data.CustomID
	parts := strings.Split(customID, ":")

	if len(parts) < 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheID := parts[1]
	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您的投稿请求已过期，请重新发起。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	emptyComponents := []discordgo.MessageComponent{}
	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    cacheData.EphChannelID,
		ID:         cacheData.EphMessageID,
		Components: &emptyComponents,
	})

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
				Content: "标题和内容都是必填的，请重新提交。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheData.RecommendTitle = strings.TrimLeft(recommendTitle, "#")
	cacheData.RecommendContent = recommendContent
	utils.UpdateCache(cacheID, cacheData)

	embed := &discordgo.MessageEmbed{
		Title:       "投稿预览",
		Description: "请检查您的安利内容，确认无误后，选择下方的提交方式。",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "安利标题", Value: recommendTitle},
			{Name: "安利内容", Value: recommendContent},
		},
		Color: 0x00BFFF,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "确认提交",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("final_submit:%s:false", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "匿名提交",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("final_submit:%s:true", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "👤"},
				},
				discordgo.Button{
					Label:    "取消",
					Style:    discordgo.DangerButton,
					CustomID: "cancel_submission",
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func FinalSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "处理您的请求时数据格式错误，请重新开始投稿。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheID := parts[1]
	isAnonymousStr := parts[2]
	isAnonymous := isAnonymousStr == "true"

	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您的投稿请求已过期，请重新发起。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	utils.RemoveFromCache(cacheID)

	content := "🍻您的安利投稿已成功提交，正在等待审核"
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: []discordgo.MessageComponent{},
			Embeds:     []*discordgo.MessageEmbed{},
		},
	})
	if err != nil {
		fmt.Printf("Error sending final response: %v\n", err)
		// As a fallback, try to edit the original interaction
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
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

	submissionID, err := utils.AddSubmissionV2(
		i.Member.User.ID, originalURL,
		cacheData.RecommendTitle, cacheData.RecommendContent,
		originalTitle, cacheData.OriginalAuthor, originalPostTimestamp, guildID, i.Member.User.Username, isAnonymous,
	)
	if err != nil {
		fmt.Printf("Error adding submission to database: %v\n", err)
		errorContent := fmt.Sprintf("提交失败，请稍后再试。错误详情: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorContent})
		return
	}

	submission := &model.Submission{
		ID:               submissionID,
		UserID:           i.Member.User.ID,
		URL:              originalURL,
		RecommendTitle:   cacheData.RecommendTitle,
		RecommendContent: cacheData.RecommendContent,
		OriginalAuthor:   cacheData.OriginalAuthor,
		IsAnonymous:      isAnonymous,
	}
	SendSubmissionToReviewChannel(s, submission)
}
