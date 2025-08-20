package bot

import (
	"amway/command"
	"amway/config"
	"amway/handler/amway"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var dg *discordgo.Session

// Start starts the bot
func Start() {
	err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading config:", err)
		return
	}

	// 注册amway处理程序
	amway.RegisterHandlers()

	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + config.Cfg.Token)
	if err != nil {
		log.Printf("error creating Discord session,", err)
		return
	}

	registerEventHandlers(dg)

	err = dg.Open()
	if err != nil {
		log.Printf("error opening connection,", err)
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
