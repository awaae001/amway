package model

import "github.com/bwmarrin/discordgo"

// DiscordPostInfo 存储解析的Discord帖子信息
type DiscordPostInfo struct {
	GuildID   string
	ChannelID string
	MessageID string
	Author    *discordgo.User
	Content   string
	Title     string
	Timestamp string
}
