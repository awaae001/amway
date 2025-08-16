package service

import (
	"context"
	"strconv"

	"amway/utils"
	recommendationPb "amway/grpc/gen/recommendation"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecommendationServiceImpl implements the RecommendationService gRPC service
type RecommendationServiceImpl struct {
	recommendationPb.UnimplementedRecommendationServiceServer
}

// NewRecommendationService creates a new instance of RecommendationServiceImpl
func NewRecommendationService() *RecommendationServiceImpl {
	return &RecommendationServiceImpl{}
}

// GetRecommendation retrieves a single recommendation by ID
func (s *RecommendationServiceImpl) GetRecommendation(ctx context.Context, req *recommendationPb.GetRecommendationRequest) (*recommendationPb.RecommendationSlip, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "推荐 ID 不能为空")
	}

	// Query the database for the submission
	submission, err := utils.GetSubmission(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询数据库失败: %v", err)
	}

	if submission == nil {
		return nil, status.Error(codes.NotFound, "未找到指定的推荐")
	}

	// Convert the submission to a RecommendationSlip
	slip, err := SubmissionToRecommendationSlip(submission)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "数据转换失败: %v", err)
	}

	return slip, nil
}

// GetRecommendationsByAuthor retrieves all recommendations by a specific author in a guild
func (s *RecommendationServiceImpl) GetRecommendationsByAuthor(ctx context.Context, req *recommendationPb.GetRecommendationsByAuthorRequest) (*recommendationPb.GetRecommendationsByAuthorResponse, error) {
	if req.AuthorId == "" {
		return nil, status.Error(codes.InvalidArgument, "作者 ID 不能为空")
	}

	if req.GuildId == 0 {
		return nil, status.Error(codes.InvalidArgument, "服务器 ID 不能为空")
	}

	// Convert guild_id from int64 to string for database query
	guildIdStr := strconv.FormatInt(req.GuildId, 10)

	// Query the database for submissions by author
	submissions, err := utils.GetSubmissionsByAuthor(req.AuthorId, guildIdStr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询数据库失败: %v", err)
	}

	// Convert submissions to RecommendationSlips
	slips, err := SubmissionsToRecommendationSlips(submissions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "数据转换失败: %v", err)
	}

	response := &recommendationPb.GetRecommendationsByAuthorResponse{
		Recommendations: slips,
	}

	return response, nil
}

// ValidateRequest performs basic validation on incoming requests
func (s *RecommendationServiceImpl) ValidateRequest(ctx context.Context) error {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return status.Error(codes.Canceled, "请求已取消")
	default:
		return nil
	}
}