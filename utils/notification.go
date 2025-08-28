package utils

import (
	"amway/model"
	"log"
)

// GlobalDiscordSession stores the Discord session for use in utils
var GlobalDiscordSession interface{}

// SendAutoRejectionDM sends a direct message to the user about automatic rejection
func SendAutoRejectionDM(submission *model.Submission, reason string) {
	// For now, just log the auto-rejection until we set up the proper session access
	log.Printf("AUTO-REJECTION NOTIFICATION: Would send DM to user %s for submission %s with reason: %s", 
		submission.UserID, submission.ID, reason)
	log.Printf("Submission details - Title: %s, Content: %s", 
		submission.RecommendTitle, submission.RecommendContent)
	
	// TODO: Implement actual DM sending when session access is properly configured
}