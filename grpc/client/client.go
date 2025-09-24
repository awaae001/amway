package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	recommendationPb "amway/grpc/gen/recommendation"
	registryPb "amway/grpc/gen/registry"
	rolePb "amway/grpc/gen/role_center"
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
	roleClient                 rolePb.RoleServiceClient
	localRecommendationService *service.RecommendationServiceImpl

	serverAddress string
	clientName    string
	token         string
	connectionID  string // 服务器分配的连接UUID

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

	// 等待响应的请求
	pendingRequests map[string]chan *registryPb.ForwardResponse
	pendingMutex    sync.RWMutex
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

		ctx:             ctx,
		cancel:          cancel,
		reconnectChan:   make(chan struct{}, 1),
		pendingRequests: make(map[string]chan *registryPb.ForwardResponse),
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
		ApiKey:  c.token,
		Address: c.clientName, // 使用客户端名称作为地址
		Services: []string{
			fmt.Sprintf("%s.RecommendationService", c.clientName), // 注册 recommendation 服务
			"role_center.RoleService", // 临时加回来测试
		},
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
				ApiKey: c.token,
				Services: []string{
					fmt.Sprintf("%s.RecommendationService", c.clientName),
				},
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

// SendGatewayRequest 通过已建立的长连接向网关发送请求
func (c *GRPCClient) SendGatewayRequest(ctx context.Context, methodPath string, requestPayload []byte, timeoutSeconds int32) (*registryPb.ForwardResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("未连接到服务器")
	}

	c.streamMutex.RLock()
	stream := c.connectionStream
	c.streamMutex.RUnlock()

	if stream == nil {
		return nil, fmt.Errorf("连接流不可用")
	}

	// 生成唯一的请求ID
	requestID := uuid.New().String()

	// 创建响应通道
	responseChan := make(chan *registryPb.ForwardResponse, 1)

	// 注册等待的请求
	c.pendingMutex.Lock()
	c.pendingRequests[requestID] = responseChan
	c.pendingMutex.Unlock()

	// 清理函数
	defer func() {
		c.pendingMutex.Lock()
		delete(c.pendingRequests, requestID)
		close(responseChan)
		c.pendingMutex.Unlock()
	}()

	// 构建请求消息
	request := &registryPb.ConnectionMessage{
		MessageType: &registryPb.ConnectionMessage_Request{
			Request: &registryPb.ForwardRequest{
				RequestId:      requestID,
				MethodPath:     methodPath,
				Headers:        make(map[string]string),
				Payload:        requestPayload,
				TimeoutSeconds: timeoutSeconds,
			},
		},
	}

	// 发送请求
	err := stream.Send(request)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	log.Printf("已发送网关请求: %s (ID: %s)", methodPath, requestID)

	// 等待响应或超时
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second // 默认超时5秒
	}

	select {
	case response := <-responseChan:
		if response == nil {
			return nil, fmt.Errorf("收到空响应")
		}
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("请求失败 (状态码: %d): %s", response.StatusCode, response.ErrorMessage)
		}
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("请求超时")
	}
}
