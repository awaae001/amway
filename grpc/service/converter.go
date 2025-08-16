package service

import (
	"strconv"

	"amway/model"
	recommendationPb "amway/grpc/gen/recommendation"
)

// SubmissionToRecommendationSlip converts a model.Submission to a RecommendationSlip protobuf message
func SubmissionToRecommendationSlip(submission *model.Submission) (*recommendationPb.RecommendationSlip, error) {
	if submission == nil {
		return nil, nil
	}

	// Convert guild_id from string to int64
	guildId, err := strconv.ParseInt(submission.GuildID, 10, 64)
	if err != nil {
		// If conversion fails, use 0 as default
		guildId = 0
	}

	return &recommendationPb.RecommendationSlip{
		Id:              submission.ID,
		AuthorId:        submission.UserID,
		AuthorNickname:  submission.AuthorNickname,
		Content:         submission.Content,
		PostUrl:         submission.URL,
		Upvotes:         int32(submission.Upvotes),
		Questions:       int32(submission.Questions),
		Downvotes:       int32(submission.Downvotes),
		CreatedAt:       submission.Timestamp,
		ReviewerId:      "", // This field is not currently tracked in the database
		IsBlocked:       false, // This would need to be computed based on is_blocked status
		GuildId:         guildId,
	}, nil
}

// SubmissionsToRecommendationSlips converts a slice of model.Submission to a slice of RecommendationSlip
func SubmissionsToRecommendationSlips(submissions []*model.Submission) ([]*recommendationPb.RecommendationSlip, error) {
	var slips []*recommendationPb.RecommendationSlip
	
	for _, submission := range submissions {
		slip, err := SubmissionToRecommendationSlip(submission)
		if err != nil {
			return nil, err
		}
		if slip != nil {
			slips = append(slips, slip)
		}
	}
	
	return slips, nil
}