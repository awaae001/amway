package amway

import (
	"amway/db"
	"amway/model"
	"amway/utils"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// RebuildCommandHandler handles the /rebuild command
func RebuildCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 立即响应交互
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral, // 仅管理员可见
		},
	})
	if err != nil {
		log.Printf("Error sending deferred response: %v", err)
		return
	}

	// 在 goroutine 中处理后续逻辑
	go func() {
		// 权限检查：只有管理员才能使用此命令
		if !utils.CheckAuth(i.Member.User.ID, i.Member.Roles) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("❌ 您没有权限执行此操作"),
			})
			return
		}

		// 获取命令参数
		options := i.ApplicationCommandData().Options
		var dryRun bool

		for _, option := range options {
			switch option.Name {
			case "dry_run":
				dryRun = option.BoolValue()
			}
		}

		// 查询需要重建的安利
		submissions, err := db.GetPendingSubmissionsWithoutMessage()
		if err != nil {
			log.Printf("Error getting pending submissions: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(fmt.Sprintf("❌ 查询安利数据时出错: %v", err)),
			})
			return
		}

		if len(submissions) == 0 {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr("✅ 没有找到需要重建的安利"),
			})
			return
		}

		// 预览模式：仅显示数量
		if dryRun {
			content := fmt.Sprintf("📊 **预览模式**\n找到 **%d** 个需要重建的安利：\n", len(submissions))
			for i, sub := range submissions {
				if i >= 10 { // 最多显示10个
					content += fmt.Sprintf("... 还有 %d 个\n", len(submissions)-10)
					break
				}
				content += fmt.Sprintf("• ID: %s (作者: <@%s>)\n", sub.ID, sub.UserID)
			}
			content += "\n使用不带 `dry_run` 参数的命令来执行重建。"

			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StringPtr(content),
			})
			return
		}

		// 实际执行重建
		successCount := 0
		failedCount := 0
		var failedIDs []string

		for _, submission := range submissions {
			if rebuildSubmissionForReview(s, submission) {
				successCount++
			} else {
				failedCount++
				failedIDs = append(failedIDs, submission.ID)
			}
		}

		// 构建结果消息
		content := fmt.Sprintf("🔄 **重建完成**\n✅ 成功重建: %d 个\n❌ 失败: %d 个\n", successCount, failedCount)

		if failedCount > 0 {
			content += fmt.Sprintf("\n失败的安利ID: %v", failedIDs)
		}

		if successCount > 0 {
			content += "\n\n重建的安利已重新发送到投票器等待审核。"
		}

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: utils.StringPtr(content),
		})
	}()
}

// rebuildSubmissionForReview 重建单个安利并发送到投票器
func rebuildSubmissionForReview(s *discordgo.Session, submission *model.Submission) bool {
	log.Printf("Rebuilding submission %s for review", submission.ID)

	// 构建 SubmissionData 用于缓存
	submissionData := model.SubmissionData{
		SubmissionID:     submission.ID,
		OriginalAuthor:   submission.OriginalAuthor,
		RecommendTitle:   submission.RecommendTitle,
		RecommendContent: submission.RecommendContent,
		ReplyToOriginal:  false, // 重建的默认不回复原帖
	}

	// 添加到缓存
	cacheID := utils.AddToCache(submissionData)

	// 使用现有的审核函数发送到审核频道
	SendSubmissionToReviewChannel(s, submission, cacheID)

	log.Printf("Successfully rebuilt and sent submission %s for review with cache ID %s", submission.ID, cacheID)
	return true
}