package main

import (
	"amway/command"
	"amway/config"
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

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Cfg.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			if i.ApplicationCommandData().Name == command.CreatePanelCommand.Name {
				amway.CreatePanelCommandHandler(s, i)
			}
		}
	})

	dg.Identify.Intents = discordgo.IntentsGuildMessages

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
