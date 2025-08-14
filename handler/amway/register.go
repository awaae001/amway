package amway

import (
	"amway/command"
	"amway/handler"
)

// RegisterHandlers registers all handlers for the amway package.
func RegisterHandlers() {
	handler.AddCommandHandler(command.CreatePanelCommand.Name, createPanelCommandHandler)
	handler.AddComponentHandler("create_submission_button", CreateSubmissionButtonHandler)

	// 新的两步投稿流程
	handler.AddModalHandler("submission_link_modal", LinkSubmissionHandler)
	handler.AddComponentHandler("confirm_post", ConfirmPostHandler)
	handler.AddComponentHandler("cancel_submission", CancelSubmissionHandler)
	handler.AddModalHandler("submission_content_modal", ContentSubmissionHandler)

	// 保留旧的处理器以确保兼容性
	handler.AddModalHandler("submission_modal", SubmissionModalHandler)

	// 审核相关处理器
	handler.AddComponentHandler("approve_submission", ApproveSubmissionHandler)
	handler.AddComponentHandler("reject_submission", RejectSubmissionHandler)
	handler.AddComponentHandler("ignore_submission", IgnoreSubmissionHandler)
	handler.AddComponentHandler("ban_submission", BanSubmissionHandler)
	handler.AddComponentHandler("delete_submission", DeleteSubmissionHandler)
}
