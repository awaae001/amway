package main

import (
	"amway/bot"
	"amway/grpc/client"
	"amway/utils"
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
	utils.InitDB()

	// 初始化 gRPC 客户端
	var grpcClient *client.GRPCClient
	if os.Getenv("GRPC_ENABLED") != "false" {
		grpcClient = client.NewGRPCClient()
		// 连接到 gRPC 服务器
		err = grpcClient.Connect()
		if err != nil {
			log.Printf("gRPC 连接失败: %v", err)
		} else {
			// 注册服务
			err = grpcClient.Register()
			if err != nil {
				log.Printf("gRPC 注册失败: %v", err)
			} else {
				// 建立反向连接
				err = grpcClient.EstablishConnection()
				if err != nil {
					log.Printf("建立反向连接失败: %v", err)
				}
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
	if grpcClient != nil && grpcClient.IsConnected() {
		grpcClient.Close()
	}
}
