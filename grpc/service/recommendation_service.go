package service

import (
	"context"
	"fmt"

	"amway/config"
	"amway/db"
	recommendationPb "amway/grpc/gen/recommendation"
	"amway/model"
	"amway/utils"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecommendationServiceImpl implements the RecommendationService gRPC service
type RecommendationServiceImpl struct {
	recommendationPb.UnimplementedRecommendationServiceServer
	discordSession *discordgo.Session
}

// NewRecommendationService creates a new instance of RecommendationServiceImpl
func NewRecommendationService() *RecommendationServiceImpl {
	return &RecommendationServiceImpl{}
}

// SetDiscordSession sets the Discord session for the service
func (s *RecommendationServiceImpl) SetDiscordSession(session *discordgo.Session) {
	s.discordSession = session
}

// GetRecommendation retrieves a single recommendation by ID
func (s *RecommendationServiceImpl) GetRecommendation(ctx context.Context, req *recommendationPb.GetRecommendationRequest) (*recommendationPb.RecommendationSlip, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "推荐 ID 不能为空")
	}

	// Query the database for the submission
	submission, err := db.GetSubmissionWithDeleted(req.Id)
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

	if req.GuildId == "" {
		return nil, status.Error(codes.InvalidArgument, "服务器 ID 不能为空")
	}

	// Query the database for submissions by author
	submissions, err := db.GetSubmissionsByAuthor(req.AuthorId, req.GuildId)
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

// CreateRecommendation creates a new recommendation submission remotely
func (s *RecommendationServiceImpl) CreateRecommendation(ctx context.Context, req *recommendationPb.CreateRecommendationRequest) (*recommendationPb.CreateRecommendationResponse, error) {
	// Validate required fields
	if req.RecommendTitle == "" {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: "推荐标题不能为空",
		}, nil
	}

	if req.RecommendContent == "" {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: "推荐内容不能为空",
		}, nil
	}

	if req.AuthorId == "" {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: "作者 ID 不能为空",
		}, nil
	}

	if req.PostUrl == "" {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: "原帖链接不能为空",
		}, nil
	}

	if req.GuildId == "" {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: "服务器 ID 不能为空",
		}, nil
	}

	// NOTE: We intentionally skip rate limiting and ban checks for remote gRPC calls
	// The calling service is responsible for these validations

	// Add submission to database
	submissionID, err := db.AddSubmissionV2(
		req.AuthorId,
		req.PostUrl,
		req.RecommendTitle,
		req.RecommendContent,
		req.OriginalTitle,
		req.OriginalAuthor,
		req.OriginalPostTimestamp,
		req.GuildId,
		req.AuthorNickname,
		req.IsAnonymous,
	)

	if err != nil {
		return &recommendationPb.CreateRecommendationResponse{
			Success: false,
			Message: fmt.Sprintf("创建失败: %v", err),
		}, nil
	}

	// Create cache entry for voting functionality
	// Extract channelID and messageID from the post URL
	var channelID, messageID string
	postInfo, err := utils.ParseDiscordURL(req.PostUrl)
	if err == nil && postInfo != nil {
		channelID = postInfo.ChannelID
		messageID = postInfo.MessageID
	}

	cacheData := model.SubmissionData{
		ChannelID:        channelID,
		MessageID:        messageID,
		OriginalAuthor:   req.OriginalAuthor,
		RecommendTitle:   req.RecommendTitle,
		RecommendContent: req.RecommendContent,
		ReplyToOriginal:  false, // Remote submissions don't reply to original
		SubmissionID:     submissionID,
	}
	cacheID := utils.AddToCache(cacheData)

	// Construct submission model for sending to review channel
	submission := &model.Submission{
		ID:               submissionID,
		UserID:           req.AuthorId,
		URL:              req.PostUrl,
		RecommendTitle:   req.RecommendTitle,
		RecommendContent: req.RecommendContent,
		OriginalAuthor:   req.OriginalAuthor,
		IsAnonymous:      req.IsAnonymous,
	}

	// Send to review channel if Discord session is available
	if s.discordSession != nil {
		// Send submission to review channel with cacheID
		err = s.sendToReviewChannel(submission, cacheID)
		if err != nil {
			fmt.Printf("发送到审核频道失败: %v，但提交已保存。提交 ID: %s\n", err, submissionID)
		}
	} else {
		// Log warning but don't fail the request
		fmt.Printf("警告: Discord Session 不可用，无法发送到审核频道。提交 ID: %s\n", submissionID)
	}

	return &recommendationPb.CreateRecommendationResponse{
		Id:      submissionID,
		Success: true,
		Message: "创建成功",
	}, nil
}

// sendToReviewChannel sends a submission to the review channel
func (s *RecommendationServiceImpl) sendToReviewChannel(submission *model.Submission, cacheID string) error {
	reviewChannelID := config.Cfg.AmwayBot.Amway.ReviewChannelID
	if reviewChannelID == "" {
		return fmt.Errorf("review channel ID not configured")
	}

	title := "新的安利投稿待审核 (远程创建)"
	if submission.IsAnonymous {
		title += " (匿名)"
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**投稿ID:** %s\n**投稿人:** <@%s>\n**安利标题:** %s\n**原帖作者:** <@%s>\n**原帖链接:** %s\n**安利内容:**\n%s", submission.ID, submission.UserID, submission.RecommendTitle, submission.OriginalAuthor, submission.URL, submission.RecommendContent),
		Color:       0xFFFF00, // Yellow for pending
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("提交时间 • ID: %s", submission.ID),
		},
	}

	// Use cacheID in button CustomIDs so voting system can find the submission
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "通过",
					Style:    discordgo.SuccessButton,
					CustomID: "vote:pass:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
				},
				discordgo.Button{
					Label:    "不通过",
					Style:    discordgo.DangerButton,
					CustomID: "vote:reject:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "❌"},
				},
				discordgo.Button{
					Label:    "封禁",
					Style:    discordgo.DangerButton,
					CustomID: "vote:ban:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🔨"},
				},
				discordgo.Button{
					Label:    "精选",
					Style:    discordgo.PrimaryButton,
					CustomID: "vote:feature:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🌟"},
				},
				discordgo.Button{
					Label:    "悔票",
					Style:    discordgo.SecondaryButton,
					CustomID: "vote:remove:" + cacheID,
					Emoji:    &discordgo.ComponentEmoji{Name: "🗑️"},
				},
			},
		},
	}

	_, err := s.discordSession.ChannelMessageSendComplex(reviewChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
	})

	if err != nil {
		return fmt.Errorf("error sending review message: %v", err)
	}

	return nil
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
