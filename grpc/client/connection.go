package client

import (
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	recommendationPb "amway/grpc/gen/recommendation"
	registryPb "amway/grpc/gen/registry"
	rolePb "amway/grpc/gen/role_center"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// 状态管理方法
func (c *GRPCClient) getConnectionState() ConnectionState {
	return ConnectionState(atomic.LoadInt32(&c.connectionState))
}

func (c *GRPCClient) setConnectionState(state ConnectionState) {
	atomic.StoreInt32(&c.connectionState, int32(state))
}

func (c *GRPCClient) IsConnected() bool {
	return c.getConnectionState() == Connected
}

func (c *GRPCClient) Connect() error {
	// 启动健康检查和重连监控
	go c.startHealthCheck()
	go c.startReconnectMonitor()

	return c.connectWithRetry()
}

func (c *GRPCClient) connectWithRetry() error {
	c.setConnectionState(Connecting)

	for attempt := 0; attempt < c.reconnectConfig.MaxRetries; attempt++ {
		log.Printf("尝试连接到 gRPC 服务器: %s (尝试 %d/%d)", c.serverAddress, attempt+1, c.reconnectConfig.MaxRetries)

		err := c.doConnect()
		if err == nil {
			c.setConnectionState(Connected)
			log.Printf("成功连接到 gRPC 服务器")

			return nil
		}

		log.Printf("连接失败: %v", err)

		if attempt < c.reconnectConfig.MaxRetries-1 {
			delay := c.calculateBackoffDelay(attempt)
			log.Printf("等待 %v 后重试...", delay)

			select {
			case <-time.After(delay):
				// 继续重试
			case <-c.ctx.Done():
				c.setConnectionState(Disconnected)
				return fmt.Errorf("连接已取消")
			}
		}
	}

	c.setConnectionState(Disconnected)
	return fmt.Errorf("达到最大重试次数，连接失败")
}

func (c *GRPCClient) doConnect() error {
	// 创建连接
	conn, err := grpc.NewClient(c.serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("无法连接到 gRPC 服务器: %v", err)
	}

	c.connectionMutex.Lock()
	defer c.connectionMutex.Unlock()

	// 关闭旧连接
	if c.conn != nil {
		c.conn.Close()
	}

	c.conn = conn
	c.registryClient = registryPb.NewRegistryServiceClient(conn)
	c.recommendationClient = recommendationPb.NewRecommendationServiceClient(conn)
	c.roleClient = rolePb.NewRoleServiceClient(conn)

	return nil
}

func (c *GRPCClient) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.reconnectConfig.BaseDelay) *
		math.Pow(c.reconnectConfig.BackoffFactor, float64(attempt)))

	if delay > c.reconnectConfig.MaxDelay {
		delay = c.reconnectConfig.MaxDelay
	}

	return delay
}

// startHealthCheck 启动连接健康检查
func (c *GRPCClient) startHealthCheck() {
	ticker := time.NewTicker(c.reconnectConfig.HealthCheckInterval)
	defer ticker.Stop()

	log.Printf("启动连接健康检查，检查间隔: %v", c.reconnectConfig.HealthCheckInterval)

	for {
		select {
		case <-ticker.C:
			if !c.checkConnectionHealth() {
				log.Printf("连接健康检查失败，触发重连")
				c.triggerReconnect()
			} else {
				// log.Printf("连接健康检查通过")
			}
		case <-c.ctx.Done():
			log.Printf("健康检查已停止")
			return
		}
	}
}

// checkConnectionHealth 检查连接健康状态
func (c *GRPCClient) checkConnectionHealth() bool {
	c.connectionMutex.RLock()
	conn := c.conn
	c.connectionMutex.RUnlock()

	if conn == nil {
		return false
	}

	state := conn.GetState()
	switch state {
	case connectivity.Ready:
		return true
	case connectivity.Connecting:
		// 连接中，暂时认为是健康的
		return true
	case connectivity.Idle:
		// 空闲状态，尝试唤醒连接
		conn.Connect()
		return true
	case connectivity.TransientFailure, connectivity.Shutdown:
		return false
	default:
		return false
	}
}

// startReconnectMonitor 启动重连监控
func (c *GRPCClient) startReconnectMonitor() {
	for {
		select {
		case <-c.reconnectChan:
			if c.getConnectionState() == Connected {
				log.Printf("开始重连流程...")
				c.performReconnect()
			}
		case <-c.ctx.Done():
			log.Printf("重连监控已停止")
			return
		}
	}
}

// triggerReconnect 触发重连
func (c *GRPCClient) triggerReconnect() {
	select {
	case c.reconnectChan <- struct{}{}:
		// 成功发送重连信号
	default:
		// 重连通道已满，说明重连正在进行中
	}
}

// performReconnect 执行重连
func (c *GRPCClient) performReconnect() {
	c.setConnectionState(Reconnecting)

	// 清除旧的连接ID
	c.connectionID = ""

	// 关闭当前连接
	c.connectionMutex.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connectionMutex.Unlock()

	// 清除连接流
	c.streamMutex.Lock()
	c.connectionStream = nil
	c.streamMutex.Unlock()

	// 尝试重新连接
	if err := c.connectWithRetry(); err != nil {
		log.Printf("重连失败: %v", err)
		c.setConnectionState(Disconnected)

		// 延迟后再次尝试重连
		go func() {
			time.Sleep(c.reconnectConfig.BaseDelay)
			c.triggerReconnect()
		}()
		return
	}

	// 重新注册和建立连接
	if err := c.Register(); err != nil {
		log.Printf("重连后注册失败: %v", err)
		c.triggerReconnect()
		return
	}

	if err := c.EstablishConnection(); err != nil {
		log.Printf("重连后建立连接失败: %v", err)
		c.triggerReconnect()
		return
	}

	log.Printf("重连成功")
}
