package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/proto"

	registryPb "amway/grpc/gen/registry"
	recommendationPb "amway/grpc/gen/recommendation"
)

func (c *GRPCClient) handleConnectionMessages(stream registryPb.RegistryService_EstablishConnectionClient) {
	heartbeatTicker := time.NewTicker(25 * time.Second)
	defer heartbeatTicker.Stop()

	defer func() {
		log.Printf("连接消息处理循环结束")
		// 如果消息处理循环结束，说明连接可能断开，触发重连
		if c.getConnectionState() == Connected {
			c.triggerReconnect()
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("连接消息处理已取消")
			return
		case <-heartbeatTicker.C:
			// 发送心跳
			heartbeatMsg := &registryPb.ConnectionMessage{
				MessageType: &registryPb.ConnectionMessage_Heartbeat{
					Heartbeat: &registryPb.Heartbeat{
						Timestamp:    time.Now().Unix(),
						ConnectionId: c.clientName,
					},
				},
			}
			if err := stream.Send(heartbeatMsg); err != nil {
				log.Printf("发送心跳失败: %v", err)
				// 发送失败可能意味着连接已断开，可以考虑触发重连
			} else {
				log.Printf("发送心跳包 -> %s", c.clientName)
			}
		default:
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("接收消息失败: %v", err)
				return
			}

			c.processMessage(msg)
		}
	}
}

func (c *GRPCClient) processMessage(msg *registryPb.ConnectionMessage) {
	c.streamMutex.RLock()
	stream := c.connectionStream
	c.streamMutex.RUnlock()

	if stream == nil {
		log.Printf("连接流不可用，无法处理消息")
		return
	}

	switch msgType := msg.MessageType.(type) {
	case *registryPb.ConnectionMessage_Request:
		// 处理转发请求
		c.handleForwardRequest(stream, msgType.Request)
	case *registryPb.ConnectionMessage_Heartbeat:
		// 处理心跳
		c.handleHeartbeat(stream, msgType.Heartbeat)
	case *registryPb.ConnectionMessage_Status:
		// 处理状态消息
		log.Printf("收到状态消息: %v", msgType.Status)
	}
}

func (c *GRPCClient) handleForwardRequest(stream registryPb.RegistryService_EstablishConnectionClient, req *registryPb.ForwardRequest) {
	log.Printf("收到转发请求: %s", req.MethodPath)

	var responsePayload []byte
	var statusCode int32 = 200
	var errorMessage string

	ctx := context.Background()

	// 根据方法路径路由到相应的服务
	switch req.MethodPath {
	case "/recommendation.RecommendationService/GetRecommendation":
		responsePayload, statusCode, errorMessage = c.handleGetRecommendation(ctx, req.Payload)
	case "/recommendation.RecommendationService/GetRecommendationsByAuthor":
		responsePayload, statusCode, errorMessage = c.handleGetRecommendationsByAuthor(ctx, req.Payload)
	default:
		statusCode = 404
		errorMessage = fmt.Sprintf("未知的方法路径: %s", req.MethodPath)
		log.Printf("未知的方法路径: %s", req.MethodPath)
	}

	// 构建响应
	response := &registryPb.ConnectionMessage{
		MessageType: &registryPb.ConnectionMessage_Response{
			Response: &registryPb.ForwardResponse{
				RequestId:    req.RequestId,
				StatusCode:   statusCode,
				Headers:      make(map[string]string),
				Payload:      responsePayload,
				ErrorMessage: errorMessage,
			},
		},
	}

	err := stream.Send(response)
	if err != nil {
		log.Printf("发送响应失败: %v", err)
	}
}

func (c *GRPCClient) handleGetRecommendation(ctx context.Context, payload []byte) ([]byte, int32, string) {
	// 解析请求
	var req recommendationPb.GetRecommendationRequest
	err := proto.Unmarshal(payload, &req)
	if err != nil {
		log.Printf("解析 GetRecommendation 请求失败: %v", err)
		return nil, 400, fmt.Sprintf("请求解析失败: %v", err)
	}

	// 调用本地服务
	resp, err := c.localRecommendationService.GetRecommendation(ctx, &req)
	if err != nil {
		log.Printf("GetRecommendation 服务调用失败: %v", err)
		return nil, 500, fmt.Sprintf("服务调用失败: %v", err)
	}

	// 序列化响应
	responsePayload, err := proto.Marshal(resp)
	if err != nil {
		log.Printf("序列化 GetRecommendation 响应失败: %v", err)
		return nil, 500, fmt.Sprintf("响应序列化失败: %v", err)
	}

	return responsePayload, 200, ""
}

func (c *GRPCClient) handleGetRecommendationsByAuthor(ctx context.Context, payload []byte) ([]byte, int32, string) {
	// 解析请求
	var req recommendationPb.GetRecommendationsByAuthorRequest
	err := proto.Unmarshal(payload, &req)
	if err != nil {
		log.Printf("解析 GetRecommendationsByAuthor 请求失败: %v", err)
		return nil, 400, fmt.Sprintf("请求解析失败: %v", err)
	}

	// 调用本地服务
	resp, err := c.localRecommendationService.GetRecommendationsByAuthor(ctx, &req)
	if err != nil {
		log.Printf("GetRecommendationsByAuthor 服务调用失败: %v", err)
		return nil, 500, fmt.Sprintf("服务调用失败: %v", err)
	}

	// 序列化响应
	responsePayload, err := proto.Marshal(resp)
	if err != nil {
		log.Printf("序列化 GetRecommendationsByAuthor 响应失败: %v", err)
		return nil, 500, fmt.Sprintf("响应序列化失败: %v", err)
	}

	return responsePayload, 200, ""
}

func (c *GRPCClient) handleHeartbeat(stream registryPb.RegistryService_EstablishConnectionClient, heartbeat *registryPb.Heartbeat) {
	receivedTime := time.Now()
	receivedTimestamp := time.Unix(heartbeat.Timestamp, 0)
	
	log.Printf("收到服务器心跳包 - ConnectionID: %s, 服务器时间: %s, 接收时间: %s, 延迟: %v", 
		heartbeat.ConnectionId,
		receivedTimestamp.Format("15:04:05"),
		receivedTime.Format("15:04:05"),
		receivedTime.Sub(receivedTimestamp))

	// 回复心跳
	responseTime := time.Now().Unix()
	response := &registryPb.ConnectionMessage{
		MessageType: &registryPb.ConnectionMessage_Heartbeat{
			Heartbeat: &registryPb.Heartbeat{
				Timestamp:    responseTime,
				ConnectionId: c.clientName,
			},
		},
	}

	err := stream.Send(response)
	if err != nil {
		log.Printf("❌ 发送心跳响应失败: %v", err)
	} else {
		log.Printf("已发送心跳响应 - ConnectionID: %s, 响应时间: %s", 
			c.clientName, 
			time.Unix(responseTime, 0).Format("15:04:05"))
	}
}