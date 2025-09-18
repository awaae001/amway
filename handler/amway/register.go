package amway

import (
	"amway/command/def"
	"amway/handler"
	amway_admin "amway/handler/amway/admin"
)

// RegisterHandlers registers all handlers for the amway package.
func RegisterHandlers() {
	handler.AddCommandHandler(def.CreatePanelCommand.Name, createPanelCommandHandler)
	handler.AddComponentHandler("create_submission_button", CreateSubmissionButtonHandler)
	handler.AddComponentHandler("how_to_submit_button", HowToSubmitButtonHandler)

	// 管理员命令处理器
	handler.AddCommandHandler(def.AmwayAdminCommand.Name, amway_admin.AmwayAdminCommandHandler)
	handler.AddCommandHandler(def.LookupCommand.Name, LookupCommandHandler)
	handler.AddCommandHandler(def.RebuildCommand.Name, RebuildCommandHandler)

	// 两步投稿流程
	handler.AddModalHandler("submission_link_modal", LinkSubmissionHandler)
	handler.AddComponentHandler("confirm_post", ConfirmPostHandler)
	handler.AddComponentHandler("cancel_submission", CancelSubmissionHandler)
	handler.AddComponentHandler("edit_submission_link", EditSubmissionLinkHandler)
	handler.AddComponentHandlerPrefix("reply_choice", ReplyChoiceHandler)
	handler.AddModalHandler("submission_content_modal", ContentSubmissionHandler)
	handler.AddComponentHandlerPrefix("edit_submission_content", EditSubmissionContentHandler)
	handler.AddComponentHandlerPrefix("confirm_preview", ConfirmPreviewHandler)
	handler.AddComponentHandlerPrefix("back_to_preview", BackToPreviewHandler)
	handler.AddComponentHandler("final_submit", FinalSubmissionHandler)

	// 审核相关处理器
	handler.AddComponentHandlerPrefix("vote:", VoteHandler)
	handler.AddModalHandler("modal_reject", ModalRejectHandler)
	handler.AddModalHandler("modal_ban", ModalBanHandler)

	// 私信通知相关处理器
	handler.AddComponentHandlerPrefix("select_reason:", SelectReasonHandler)
	handler.AddComponentHandlerPrefix("send_rejection_dm:", SendRejectionDMHandler)
	handler.AddComponentHandlerPrefix("select_ban_reason:", SelectBanReasonHandler)
	handler.AddComponentHandlerPrefix("send_ban_dm:", SendBanDMHandler)
}
