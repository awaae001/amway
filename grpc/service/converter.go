package service

import (
	recommendationPb "amway/grpc/gen/recommendation"
	"amway/model"
)

// SubmissionToRecommendationSlip converts a model.Submission to a RecommendationSlip protobuf message
func SubmissionToRecommendationSlip(submission *model.Submission) (*recommendationPb.RecommendationSlip, error) {
	if submission == nil {
		return nil, nil
	}

	return &recommendationPb.RecommendationSlip{
		Id:                    submission.ID,
		AuthorId:              submission.UserID,
		AuthorNickname:        submission.AuthorNickname,
		Content:               submission.Content,
		PostUrl:               submission.URL,
		Upvotes:               int32(submission.Upvotes),
		Questions:             int32(submission.Questions),
		Downvotes:             int32(submission.Downvotes),
		CreatedAt:             submission.Timestamp,
		ReviewerId:            "", // This field is not currently tracked in the database
		GuildId:               submission.GuildID,
		OriginalTitle:         submission.OriginalTitle,
		OriginalAuthor:        submission.OriginalAuthor,
		RecommendTitle:        submission.RecommendTitle,
		RecommendContent:      submission.RecommendContent,
		OriginalPostTimestamp: submission.OriginalPostTimestamp,
		FinalAmwayMessageId:   submission.FinalAmwayMessageID,
		IsDeleted:             false, // Assuming default is false as it's not in model
		IsAnonymous:           submission.IsAnonymous,
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
