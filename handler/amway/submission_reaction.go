package amway

import (
	"amway/config"
	"amway/db"
	"amway/model"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// MessageReactionAdd handles reaction additions.
func MessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID || r.ChannelID != config.Cfg.AmwayBot.Amway.PublishChannelID || !isValidReaction(r.Emoji.Name) {
		return
	}
	handleReactionUpdate(s, r.ChannelID, r.MessageID, r.UserID, r.Emoji.Name, "ADD")
}

// MessageReactionRemove handles reaction removals.
func MessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID || r.ChannelID != config.Cfg.AmwayBot.Amway.PublishChannelID || !isValidReaction(r.Emoji.Name) {
		return
	}
	handleReactionUpdate(s, r.ChannelID, r.MessageID, r.UserID, r.Emoji.Name, "REMOVE")
}

func handleReactionUpdate(s *discordgo.Session, channelID, messageID, userID, emojiName, action string) {
	submission, err := db.GetSubmissionByMessageID(messageID)
	if err != nil {
		log.Printf("Error getting submission by message ID %s: %v", messageID, err)
		return
	}
	if submission == nil {
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	oldReaction, err := db.GetReaction(submission.ID, userID)
	if err != nil {
		log.Printf("Error getting reaction for submission %s, user %s: %v", submission.ID, userID, err)
		return
	}

	var emojiToRemove string

	switch action {
	case "ADD":
		if oldReaction != nil && oldReaction.EmojiName == emojiName {
			return // User reacted with the same emoji again, do nothing.
		}

		if oldReaction != nil {
			if err := db.UpdateReactionCountInTx(tx, submission.ID, oldReaction.EmojiName, -1); err != nil {
				log.Printf("Error decrementing old reaction count: %v", err)
				return
			}
			emojiToRemove = oldReaction.EmojiName
		}

		if err := db.UpdateReactionCountInTx(tx, submission.ID, emojiName, 1); err != nil {
			log.Printf("Error incrementing new reaction count: %v", err)
			return
		}

		newReaction := &model.SubmissionReaction{
			SubmissionID: submission.ID,
			MessageID:    messageID,
			UserID:       userID,
			EmojiName:    emojiName,
			CreatedAt:    time.Now().Unix(),
		}
		if err := db.UpsertReactionInTx(tx, newReaction); err != nil {
			log.Printf("Error upserting reaction: %v", err)
			return
		}

	case "REMOVE":
		if oldReaction == nil || oldReaction.EmojiName != emojiName {
			return
		}

		if err := db.UpdateReactionCountInTx(tx, submission.ID, emojiName, -1); err != nil {
			log.Printf("Error decrementing removed reaction count: %v", err)
			return
		}

		if err := db.DeleteReactionInTx(tx, submission.ID, userID); err != nil {
			log.Printf("Error deleting reaction: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	// After the transaction is successfully committed, remove the old reaction from the message.
	if emojiToRemove != "" {
		err := s.MessageReactionRemove(channelID, messageID, emojiToRemove, userID)
		if err != nil {
			// This is not a critical error, just log it. The database is already correct.
			log.Printf("Failed to remove old reaction emoji '%s' for user %s on message %s: %v", emojiToRemove, userID, messageID, err)
		}
	}
}

func isValidReaction(emojiName string) bool {
	switch emojiName {
	case "👍", "🤔", "🚫":
		return true
	default:
		return false
	}
}
