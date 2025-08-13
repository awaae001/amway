package main

import (
	"amway/command"
	"amway/config"
	"amway/handler"
	"amway/handler/amway"
	"amway/utils"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	utils.InitDB()

	err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// 注册amway处理程序
	amway.RegisterHandlers()

	// 调试信息：检查token是否被正确读取
	if config.Cfg.Token == "" {
		fmt.Println("Warning: Token is empty!")
	} else {
		fmt.Printf("Token loaded successfully (length: %d)\n", len(config.Cfg.Token))
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Cfg.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(handler.OnInteractionCreate)

	// 设置必要的intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	for _, guildID := range config.Cfg.Commands.Allowguils {
		_, err := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, command.CreatePanelCommand)
		if err != nil {
			log.Fatalf("Cannot create command: %v", err)
		}
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
