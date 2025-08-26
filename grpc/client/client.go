package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	recommendationPb "amway/grpc/gen/recommendation"
	registryPb "amway/grpc/gen/registry"
	"amway/grpc/service"
)

type ConnectionState int32

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
	Reconnecting
)

type ReconnectConfig struct {
	MaxRetries          int
	BaseDelay           time.Duration
	MaxDelay            time.Duration
	BackoffFactor       float64
	HealthCheckInterval time.Duration
}

type GRPCClient struct {
	conn                       *grpc.ClientConn
	registryClient             registryPb.RegistryServiceClient
	recommendationClient       recommendationPb.RecommendationServiceClient
	localRecommendationService *service.RecommendationServiceImpl

	serverAddress string
	clientName    string
	token         string

	// 连接状态管理
	connectionState int32 // 使用 atomic 操作
	connectionMutex sync.RWMutex

	// 重连配置
	reconnectConfig ReconnectConfig

	// 控制通道
	ctx           context.Context
	cancel        context.CancelFunc
	reconnectChan chan struct{}

	// 连接流
	connectionStream registryPb.RegistryService_EstablishConnectionClient
	streamMutex      sync.RWMutex
}

func NewGRPCClient() *GRPCClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCClient{
		serverAddress:              os.Getenv("GRPC_SERVER_ADDRESS"),
		clientName:                 os.Getenv("GRPC_CLIENT_NAME"),
		token:                      os.Getenv("GRPC_TOKEN"),
		localRecommendationService: service.NewRecommendationService(),

		connectionState: int32(Disconnected),
		reconnectConfig: ReconnectConfig{
			MaxRetries:          10,
			BaseDelay:           2 * time.Second,
			MaxDelay:            60 * time.Second,
			BackoffFactor:       2.0,
			HealthCheckInterval: 30 * time.Second,
		},

		ctx:           ctx,
		cancel:        cancel,
		reconnectChan: make(chan struct{}, 1),
	}
}
func (c *GRPCClient) Start() {
	go func() {
		if err := c.Connect(); err != nil {
			log.Fatalf("无法连接到 gRPC 服务器: %v", err)
		}

		if err := c.Register(); err != nil {
			log.Fatalf("客户端注册失败: %v", err)
		}

		if err := c.EstablishConnection(); err != nil {
			log.Fatalf("建立反向连接失败: %v", err)
		}
	}()
}

func (c *GRPCClient) Register() error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接到服务器")
	}

	log.Printf("正在注册客户端: %s", c.clientName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 注册请求
	req := &registryPb.RegisterRequest{
		ApiKey:   c.token,
		Address:  c.clientName,                                                    // 使用客户端名称作为地址
		Services: []string{fmt.Sprintf("%s.RecommendationService", c.clientName)}, // 注册 recommendation 服务
	}

	resp, err := c.registryClient.Register(ctx, req)
	if err != nil {
		return fmt.Errorf("注册失败: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("注册被拒绝: %s", resp.Message)
	}

	log.Printf("客户端注册成功: %s", resp.Message)
	return nil
}

func (c *GRPCClient) EstablishConnection() error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接到服务器")
	}

	log.Printf("建立反向连接...")

	ctx := context.Background()
	stream, err := c.registryClient.EstablishConnection(ctx)
	if err != nil {
		return fmt.Errorf("建立连接失败: %v", err)
	}

	// 发送连接注册消息
	registerMsg := &registryPb.ConnectionMessage{
		MessageType: &registryPb.ConnectionMessage_Register{
			Register: &registryPb.ConnectionRegister{
				ApiKey:   c.token,
				Services: []string{fmt.Sprintf("%s.RecommendationService", c.clientName)},
			},
		},
	}

	err = stream.Send(registerMsg)
	if err != nil {
		return fmt.Errorf("发送注册消息失败: %v", err)
	}

	log.Printf("反向连接建立成功")

	// 保存连接流
	c.streamMutex.Lock()
	c.connectionStream = stream
	c.streamMutex.Unlock()

	// 启动消息处理循环
	go c.handleConnectionMessages(stream)

	return nil
}

func (c *GRPCClient) Close() error {
	c.cancel() // 取消所有 goroutine

	c.connectionMutex.Lock()
	defer c.connectionMutex.Unlock()

	if c.conn != nil {
		log.Printf("关闭 gRPC 连接")
		err := c.conn.Close()
		c.conn = nil
		c.setConnectionState(Disconnected)
		return err
	}
	return nil
}

// RecommendationService 相关方法
func (c *GRPCClient) GetRecommendation(id string) (*recommendationPb.RecommendationSlip, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("未连接到服务器")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &recommendationPb.GetRecommendationRequest{
		Id: id,
	}

	return c.recommendationClient.GetRecommendation(ctx, req)
}

func (c *GRPCClient) GetRecommendationsByAuthor(authorId string, guildId string) (*recommendationPb.GetRecommendationsByAuthorResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("未连接到服务器")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &recommendationPb.GetRecommendationsByAuthorRequest{
		AuthorId: authorId,
		GuildId:  guildId,
	}

	return c.recommendationClient.GetRecommendationsByAuthor(ctx, req)
}
