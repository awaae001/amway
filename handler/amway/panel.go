package amway

import (
	"amway/command"
	"amway/config"
	"amway/handler"
	"amway/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	handler.AddCommandHandler(command.CreatePanelCommand.Name, createPanelCommandHandler)
}

func createPanelCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// 检查权限
	if !utils.CheckAuth(i.Member.User.ID, i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "你没有权限执行此操作。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	channelID := config.Cfg.AmwayBot.Amway.PublishChannelID

	embed := &discordgo.MessageEmbed{
		Title:       "鉴赏家投稿面板",
		Description: "点击下方按钮开始投稿您的简评",
	}

	button := discordgo.Button{
		Label:    "点击投稿",
		Style:    discordgo.PrimaryButton,
		CustomID: "create_submission_button", // This CustomID will be used by the submission handler
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "面板已创建",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{button},
			},
		},
	})

	if err != nil {
		fmt.Println("Error sending panel message:", err)
	}
}
