package amway

import (
	"amway/model"
	"amway/utils"
	"bytes"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

// BuildSubmissionLinkModal 创建并返回用于提交帖子链接的模态框
func BuildSubmissionLinkModal() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "submission_link_modal",
			Title:    "步骤 1/6：提供帖子链接",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "submission_url",
							Label:       "Discord帖子链接",
							Style:       discordgo.TextInputShort,
							Placeholder: "请输入要安利的Discord帖子链接（复制消息链接）",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// BuildPostConfirmationComponents 创建并返回帖子信息确认的 Embed 和按钮
func BuildPostConfirmationComponents(postInfo *model.DiscordPostInfo) ([]*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	postInfoText := utils.FormatDiscordPostInfo(postInfo)
	embed := &discordgo.MessageEmbed{
		Title:       "步骤 2/6：确认帖子信息",
		Description: fmt.Sprintf("**已识别的帖子信息：**\n%s\n\n请仔细确认以上信息无误，然后选择继续下一步", postInfoText),
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
					Label:    "编辑链接",
					Style:    discordgo.SecondaryButton,
					CustomID: "edit_submission_link",
					Emoji:    &discordgo.ComponentEmoji{Name: "✏️"},
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

	return []*discordgo.MessageEmbed{embed}, components
}

// BuildReplyChoiceComponents 创建并返回一个嵌入式消息和一组按钮，用于用户选择是否回复原帖
func BuildReplyChoiceComponents(cacheID string) ([]*discordgo.MessageEmbed, []discordgo.MessageComponent) {
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

	embed := &discordgo.MessageEmbed{
		Title:       "步骤 3/6：选择回复方式",
		Description: "**请选择投稿发布方式：**\n\n• **发送到原帖**：您的安利将作为回复出现在原帖下方\n• **仅投稿**：您的安利仅在安利频道发布\n\n💡 建议选择发送到原帖，让原作者知道您的推荐！",
		Color:       0x0099ff, // A nice blue color
	}
	return []*discordgo.MessageEmbed{embed}, components
}

// BuildSubmissionContentModal 创建并返回用于提交安利内容的模态框
func BuildSubmissionContentModal(cacheID string, title, content string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("submission_content_modal:%s", cacheID),
			Title:    "步骤 4/6：编写安利内容",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_title",
							Label:       "安利标题",
							Style:       discordgo.TextInputShort,
							Placeholder: "用一句话概括您的安利亮点（将以粗体显示）",
							Required:    true,
							Value:       title,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "recommend_content",
							Label:       "安利内容",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "详细说明推荐理由，分享您的感受和见解（20-1024字）",
							Required:    true,
							MinLength:   20,
							MaxLength:   1024,
							Value:       content,
						},
					},
				},
			},
		},
	}
}

// BuildSubmissionPreviewComponents 创建并返回投稿预览的 Embed 和按钮
func BuildSubmissionPreviewComponents(recommendTitle, recommendContent, cacheID string) ([]*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "步骤 5/6：预览安利内容",
		Description: "**请仔细检查您的安利内容：**\n\n确认无误后，请点击下方按钮继续到最后一步",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "安利标题", Value: fmt.Sprintf("**%s**", recommendTitle)},
			{Name: "安利内容", Value: recommendContent},
		},
		Color: 0x00BFFF,
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "确认内容，继续下一步",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_preview:%s", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "▶️"},
				},
				discordgo.Button{
					Label:    "编辑内容",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("edit_submission_content:%s", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "✏️"},
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

	return []*discordgo.MessageEmbed{embed}, components
}

// BuildCancelResponseData 创建并返回一个用于表示投稿已取消的响应数据
func BuildCancelResponseData() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Content:    "投稿已取消",
		Components: []discordgo.MessageComponent{},
		Embeds:     []*discordgo.MessageEmbed{},
	}
}

// BuildFinalSuccessResponseData 创建并返回最终成功提交的响应数据
func BuildFinalSuccessResponseData() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Content:    "🍻您的安利投稿已成功提交，正在等待审核",
		Components: []discordgo.MessageComponent{},
		Embeds:     []*discordgo.MessageEmbed{},
		Flags:      discordgo.MessageFlagsEphemeral,
	}
}

// BuildHowToSubmitResponseData 创建并返回投稿指南的响应数据
func BuildHowToSubmitResponseData() *discordgo.InteractionResponseData {
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
	return responseData
}

// BuildAnonymityChoiceComponents 创建并返回独立的匿名选择界面
func BuildAnonymityChoiceComponents(cacheID string) ([]*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "步骤 6/6：选择提交方式",
		Description: "**请选择您的投稿提交方式：**\n\n" +
			"**实名提交**：您的Discord用户名将显示在安利中\n" +
			"**匿名提交**：您的用户名不会显示，保护您的隐私\n\n" +
			"💡 **提示**：匿名提交后仍可在「我的安利」中管理您的投稿",
		Color: 0xFF9500, // Orange color to make it stand out
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "关于匿名投稿",
				Value:  "• 您的Discord用户名不会在发布的安利中显示\n• 管理员仍能看到您的身份以便联系\n• 投稿后可在个人面板中切换匿名状态",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "这是投稿的最后一步，请谨慎选择！",
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "实名提交",
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
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "返回上一步",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("back_to_preview:%s", cacheID),
					Emoji:    &discordgo.ComponentEmoji{Name: "◀️"},
				},
				discordgo.Button{
					Label:    "取消投稿",
					Style:    discordgo.DangerButton,
					CustomID: "cancel_submission",
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
			},
		},
	}

	return []*discordgo.MessageEmbed{embed}, components
}
