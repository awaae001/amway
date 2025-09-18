package bot

import (
	"amway/command"
	"amway/config"
	"amway/handler/amway"
	"amway/handler/my"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var dg *discordgo.Session

// Start 启动机器人
func Start() {
	err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置文件时出错: %v", err)
		return
	}

	// 注册 amway 处理程序
	amway.RegisterHandlers()
	my.RegisterHandlers()

	// 使用提供的机器人令牌创建一个新的 Discord 会话
	dg, err = discordgo.New("Bot " + config.Cfg.Token)
	if err != nil {
		log.Printf("创建 Discord 会话时出错, %v", err)
		return
	}

	registerEventHandlers(dg)

	err = dg.Open()
	if err != nil {
		log.Printf("error opening connection, %v", err)
		return
	}

	for _, guildID := range config.Cfg.Commands.Allowguils {
		for _, cmd := range command.AllCommands {
			_, err := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, cmd)
			if err != nil {
				log.Fatalf("Cannot create '%v' command: %v", cmd.Name, err)
			}
		}
	}

	log.Printf("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

// GetSession 返回当前的 Discord 会话
func GetSession() *discordgo.Session {
	return dg
}
