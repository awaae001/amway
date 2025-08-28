package amway

import (
	"amway/db"
	"amway/model"
	"amway/utils"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func CreateSubmissionButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	banned, err := db.IsUserBanned(i.Member.User.ID)
	if err != nil {
		fmt.Printf("Error checking if user is banned: %v\n", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "无法处理您的请求，请稍后再试",
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

	postInfoText := utils.FormatDiscordPostInfo(postInfo)
	embed := &discordgo.MessageEmbed{
		Title:       "帖子信息确认",
		Description: fmt.Sprintf("%s\n\n请确认以上信息无误，然后点击下方按钮继续填写安利内容", postInfoText),
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
				Content: "数据格式错误，请重新开始投稿流程",
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
	}
	cacheID := utils.AddToCache(cacheData)

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "是，发送到原帖",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("reply_choice:%s:true", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "否，仅投稿",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("reply_choice:%s:false", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "请选择：是否将您的安利作为回复发送到原帖下方？",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
			Embeds:     []*discordgo.MessageEmbed{},
		},
	})

	if err != nil {
		fmt.Printf("Error updating message for reply choice: %v\n", err)
	}
}

func ReplyChoiceHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "数据格式错误，请重新开始投稿流程",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheID := parts[1]
	replyToOriginalStr := parts[2]
	replyToOriginal := replyToOriginalStr == "true"

	cacheData, found := utils.GetFromCache(cacheID)
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "您的投稿请求已过期，请重新发起",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cacheData.ReplyToOriginal = replyToOriginal
	utils.UpdateCache(cacheID, cacheData)

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
		fmt.Printf("Error creating content modal after reply choice: %v\n", err)
	}
}
func CancelSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "投稿已取消",
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
				Content: "数据格式错误，请重新开始投稿流程",
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
				Content: "您的投稿请求已过期，请重新发起",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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

	embed := &discordgo.MessageEmbed{
		Title:       "投稿预览",
		Description: "请检查您的安利内容，确认无误后，选择下方的提交方式",
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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error updating message with preview: %v\n", err)
	}
}

func FinalSubmissionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	banned, err := db.IsUserBanned(i.Member.User.ID)
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

	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "处理您的请求时数据格式错误，请重新开始投稿",
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
				Content: "您的投稿请求已过期，请重新发起",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	utils.RemoveFromCache(cacheID)

	finalContent := "🍻您的安利投稿已成功提交，正在等待审核"
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    finalContent,
			Components: []discordgo.MessageComponent{},
			Embeds:     []*discordgo.MessageEmbed{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
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
	SendSubmissionToReviewChannel(s, submission)
}

func HowToSubmitButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guideContent := `### 欢迎你！为了让每一份安利都能闪闪发光，也为了让审核流程更顺畅，请花几分钟阅读这份投稿指南

## 第一步：找到你想要安利的帖子
- 定位帖子：在 Discord 的任意公开频道中，找到你想要安利的原帖
- 复制链接：右键点击该帖子（或长按，如果你在手机上），选择“复制消息链接”

## 第二步：开始投稿
- 点击按钮：回到我们的投稿面板，点击“点击投稿”按钮
- 粘贴链接：在弹出的第一个窗口中，将你刚刚复制的帖子链接粘贴进去，然后点击提交

## 第三步：填写安利内容
- 确认信息：机器人会自动抓取帖子的基本信息，请你核对一遍，确保无误后点击“确认并继续”
- 撰写安利：在第二个窗口中，你需要填写两个部分：
    1. 安利标题：用一句话概括你的安利亮点，它会以加粗大字的形式显示
   2. 安利内容：详细说明你的推荐理由，分享你的感受和见解我们鼓励真诚、有深度的分享，字数建议在 20 到 1024 字之间
- 预览与提交：填写完毕后，你会看到最终的预览效果在这里，你可以选择**实名提交**或**匿名提交**

## 重要须知
- 关于匿名：选择匿名提交后，你的 Discord 用户名将不会在最终发布的安利中显示
- 内容审核：所有提交的安利都会进入审核队列，由管理组进行审阅请确保你的内容友好、尊重原创，并且不包含不适宜的言论
- 身份组奖励：当你的历史投稿累计达到 5 条并通过审核后，你将有资格申请专属的 <@&1376078089024573570> 身份组，以表彰你对社区的贡献！

## 遇到问题？
如果在投稿过程中遇到任何困难，或者对流程有任何疑问，<@&1337441650137366705> ，维护组会转接开发者

感谢你的分享，期待看到你的精彩安利！`
	embed := &discordgo.MessageEmbed{
		Title:       "抚松安利小助手 · 投稿指南",
		Description: guideContent,
		Color:       0x5865F2, // Discord Blurple
	}

	responseData := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	}

	image, err := os.ReadFile("src/bgimage.webp")
	if err == nil {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "attachment://bgimage.webp",
		}
		responseData.Files = []*discordgo.File{
			{
				Name:        "bgimage.webp",
				ContentType: "image/webp",
				Reader:      bytes.NewReader(image),
			},
		}
	} else {
		fmt.Printf("Error reading image file, sending embed without image: %v\n", err)
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
	})

	if err != nil {
		fmt.Printf("Error sending how-to-submit embed: %v\n", err)
	}
}
