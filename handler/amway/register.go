package amway

import (
	"amway/command"
	"amway/handler"
)

// RegisterHandlers registers all handlers for the amway package.
func RegisterHandlers() {
	handler.AddCommandHandler(command.CreatePanelCommand.Name, createPanelCommandHandler)
	handler.AddComponentHandler("create_submission_button", createSubmissionButtonHandler)
	handler.AddModalHandler("submission_modal", submissionModalHandler)
	handler.AddComponentHandler("approve_submission", approveSubmissionHandler)
	handler.AddComponentHandler("reject_submission", rejectSubmissionHandler)
	handler.AddComponentHandler("ignore_submission", ignoreSubmissionHandler)
	handler.AddComponentHandler("ban_submission", banSubmissionHandler)
	handler.AddComponentHandler("delete_submission", deleteSubmissionHandler)
}
