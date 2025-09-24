package amway

import (
	"amway/shared"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func TestAssignRoleHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	configID := optionMap["config_id"].StringValue()
	user := optionMap["user"].UserValue(s)
	guildID := i.GuildID

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "正在尝试分配身份组...",
		},
	})

	success, err := shared.GRPCClient.AssignRole(guildID, configID, user.ID)
	var content string
	if err != nil {
		content = fmt.Sprintf("为用户 %s 分配身份组失败: %v", user.Mention(), err)
		log.Printf("为用户 %s 分配身份组失败: %v", user.ID, err)
	} else if success {
		content = fmt.Sprintf("成功为用户 %s 分配身份组 (config: %s)", user.Mention(), configID)
		log.Printf("成功为用户 %s 分配身份组", user.ID)
	} else {
		content = fmt.Sprintf("为用户 %s 分配身份组未成功，但没有错误返回", user.Mention())
		log.Printf("为用户 %s 分配身份组未成功，但没有错误返回", user.ID)
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}
