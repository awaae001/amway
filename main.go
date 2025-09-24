//go:generate mkdir -p grpc/gen/registry grpc/gen/recommendation
//go:generate protoc --go_out=grpc/gen/registry --go_opt=paths=source_relative --go-grpc_out=grpc/gen/registry --go-grpc_opt=paths=source_relative --proto_path=grpc/proto grpc/proto/registry.proto
//go:generate protoc --go_out=grpc/gen/recommendation --go_opt=paths=source_relative --go-grpc_out=grpc/gen/recommendation --go-grpc_opt=paths=source_relative --proto_path=doc/proto doc/proto/recommendation.proto

package main

import (
	"amway/bot"
	"amway/db"
	"amway/grpc/client"
	"amway/shared"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Printf("警告: 无法加载 .env 文件: %v", err)
	}

	// 初始化数据库
	db.InitDB()

	// 初始化 gRPC 客户端
	if os.Getenv("GRPC_ENABLED") != "false" {
		shared.GRPCClient = client.NewGRPCClient()
		// 连接到 gRPC 服务器
		err = shared.GRPCClient.Connect()
		if err != nil {
			log.Printf("gRPC 连接失败: %v", err)
		} else {
			// 直接建立反向连接（包含注册逻辑）
			err = shared.GRPCClient.EstablishConnection()
			if err != nil {
				log.Printf("建立反向连接失败: %v", err)
			}
		}
	}

	// 启动 Discord 机器人
	bot.Start()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("正在关闭...")

	// 关闭 gRPC 连接
	if shared.GRPCClient != nil && shared.GRPCClient.IsConnected() {
		shared.GRPCClient.Close()
	}
}
