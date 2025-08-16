package amway

import (
	"amway/config"
	"amway/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// MessageReactionAdd handles reaction additions.
func MessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if the reaction is in the publish channel
	if r.ChannelID != config.Cfg.AmwayBot.Amway.PublishChannelID {
		return
	}

	handleReaction(r.MessageReaction, 1)
}

// MessageReactionRemove handles reaction removals.
func MessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if the reaction is in the publish channel
	if r.ChannelID != config.Cfg.AmwayBot.Amway.PublishChannelID {
		return
	}

	handleReaction(r.MessageReaction, -1)
}

func handleReaction(r *discordgo.MessageReaction, increment int) {
	submission, err := utils.GetSubmissionByMessageID(r.MessageID)
	if err != nil {
		return
	}

	if submission == nil {
		return
		fmt.Println("Error: Submission not found")
	}

	err = utils.UpdateReactionCount(submission.ID, r.Emoji.Name, increment)
	if err != nil {
		fmt.Printf("Error updating reaction count for submission %s: %v\n", submission.ID, err)
	}
}
