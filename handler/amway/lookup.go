package amway

import (
	"amway/db"
	"amway/model"
	"amway/utils"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	submissionsPerPage = 5
	maxPages           = 20
)

// LookupCommandHandler handles the /lookup command
func LookupCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 立即响应交互
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral, // 结果仅对用户可见
		},
	})
	if err != nil {
		log.Printf("Error sending deferred response: %v", err)
		return
	}

	// 在 goroutine 中处理后续逻辑
	go func() {
		// 1. 解析参数
		options := i.ApplicationCommandData().Options
		var targetUser *discordgo.User
		if len(options) > 0 && options[0].Name == "user" {
			targetUser = options[0].UserValue(s)
		} else {
			targetUser = i.Member.User
		}

		// 2. 从数据库获取投稿
		submissions, err := db.GetAllSubmissionsByAuthor(targetUser.ID)
		if err != nil {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("❌ 查询投稿失败: %v", err)),
			})
			return
		}

		// 3. 根据规则过滤投稿
		var filteredSubmissions []*model.Submission
		isQueryingSelf := i.Member.User.ID == targetUser.ID
		for _, sub := range submissions {
			if !isQueryingSelf && sub.IsAnonymous {
				continue // 如果查询他人，则跳过匿名投稿
			}
			filteredSubmissions = append(filteredSubmissions, sub)
		}

		// 4. 检查结果
		if len(filteredSubmissions) == 0 {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("ℹ️ 未找到用户 <@%s> 的任何投稿", targetUser.ID)),
			})
			return
		}

		// 5. 分页并发送结果
		sendPaginatedSubmissions(s, i, targetUser, filteredSubmissions, 0)
	}()
}

// sendPaginatedSubmissions 发送分页的投稿列表
func sendPaginatedSubmissions(s *discordgo.Session, i *discordgo.InteractionCreate, targetUser *discordgo.User, submissions []*model.Submission, page int) {
	start := page * submissionsPerPage
	end := start + submissionsPerPage
	if end > len(submissions) {
		end = len(submissions)
	}

	totalPages := (len(submissions) + submissionsPerPage - 1) / submissionsPerPage
	if totalPages > maxPages {
		totalPages = maxPages
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("👤 %s 的投稿历史", targetUser.Username),
		Description: fmt.Sprintf("共找到 %d 条投稿正在显示第 %d / %d 页", len(submissions), page+1, totalPages),
		Color:       0x5865F2, // Discord Blurple
		Fields:      []*discordgo.MessageEmbedField{},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Amway Bot",
		},
	}

	for _, sub := range submissions[start:end] {
		title := sub.RecommendTitle
		if title == "" {
			title = "无标题"
		}
		if sub.IsAnonymous {
			title += " (匿名)"
		}

		contentPreview := sub.RecommendContent
		if len(contentPreview) > 100 {
			contentPreview = string([]rune(contentPreview)[:100]) + "..."
		}

		// 将 Discord 时间戳转换为更易读的格式
		timestamp := time.Unix(sub.Timestamp, 0).Format("2006-01-02")
		link := fmt.Sprintf("https://discord.com/channels/%s/%s", sub.GuildID, sub.FinalAmwayMessageID)

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("`%s` | %s", sub.ID, title),
			Value: fmt.Sprintf("[链接](%s) • %s • 👍 %d ✅ %d  ❌ %d\n> %s", link, timestamp, sub.Upvotes, sub.Questions, sub.Downvotes, contentPreview),
		})
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "上一页",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("lookup_prev:%d:%s", page, targetUser.ID),
					Disabled: page == 0,
				},
				discordgo.Button{
					Label:    "下一页",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("lookup_next:%d:%s", page, targetUser.ID),
					Disabled: end >= len(submissions) || page >= maxPages-1,
				},
			},
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}
