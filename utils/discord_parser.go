package utils

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"

	"amway/model"
)

// ParseDiscordURL 解析Discord链接（支持频道链接和消息链接）
func ParseDiscordURL(url string) (*model.DiscordPostInfo, error) {
	reMessage := regexp.MustCompile(`https://discord(?:app)?\.com/channels/(\d+)/(\d+)/(\d+)`)
	matches := reMessage.FindStringSubmatch(url)

	if len(matches) == 4 {
		return &model.DiscordPostInfo{
			GuildID:   matches[1],
			ChannelID: matches[2],
			MessageID: matches[3],
		}, nil
	}
	reChannel := regexp.MustCompile(`https://discord(?:app)?\.com/channels/(\d+)/(\d+)/?$`)
	matches = reChannel.FindStringSubmatch(url)

	if len(matches) == 3 {
		return &model.DiscordPostInfo{
			GuildID:   matches[1],
			ChannelID: matches[2],
			MessageID: "", // 需要后续获取首楼消息ID
		}, nil
	}

	return nil, errors.New("无效的Discord链接格式")
}

// FetchDiscordMessage 从Discord API获取消息详细信息
func FetchDiscordMessage(s *discordgo.Session, info *model.DiscordPostInfo) error {
	var message *discordgo.Message
	var err error

	if info.MessageID == "" {
		// 如果没有具体消息ID，获取频道/线程信息并获取首楼消息
		thread, err := s.Channel(info.ChannelID)
		if err != nil {
			return fmt.Errorf("无法获取频道信息: %v", err)
		}

		// 获取首楼消息（线程ID就是首楼消息ID）
		firstMessage, err := s.ChannelMessage(thread.ID, thread.ID)
		if err != nil {
			return fmt.Errorf("无法获取首楼消息: %v", err)
		}

		message = firstMessage
		info.MessageID = thread.ID
		info.Title = thread.Name
	} else {
		// 直接获取指定消息
		message, err = s.ChannelMessage(info.ChannelID, info.MessageID)
		if err != nil {
			return fmt.Errorf("无法获取消息: %v", err)
		}
		// 即使是具体消息，也尝试获取其所在频道的标题
		thread, err := s.Channel(info.ChannelID)
		if err != nil {
			// 如果获取频道信息失败，标题将为空，但流程继续
			info.Title = ""
		} else {
			info.Title = thread.Name
		}
	}

	info.Author = message.Author
	info.Content = message.Content
	info.Timestamp = message.Timestamp.Format("2006-01-02 15:04:05")

	return nil
}

// ValidateDiscordPost 验证Discord帖子信息
func ValidateDiscordPost(s *discordgo.Session, url, currentGuildID, submitterUserID string) (*model.DiscordPostInfo, error) {
	// 解析链接
	info, err := ParseDiscordURL(url)
	if err != nil {
		return nil, err
	}

	// 验证是否为当前服务器
	if info.GuildID != currentGuildID {
		return nil, errors.New("只能安利本服务器内的帖子")
	}

	// 获取消息详细信息
	err = FetchDiscordMessage(s, info)
	if err != nil {
		return nil, err
	}

	// 验证不能安利自己发布的帖子
	if info.Author.ID == submitterUserID {
		return nil, errors.New("不能安利自己发布的帖子")
	}

	return info, nil
}

// FormatDiscordPostInfo 格式化Discord帖子信息用于展示
func FormatDiscordPostInfo(info *model.DiscordPostInfo) string {
	content := info.Content
	if len(content) > 200 {
		content = content[:200] + "..."
	}

	title := info.Title
	if title == "" {
		title = "无标题"
	}

	return fmt.Sprintf(
		"**帖子标题:** %s\n**原帖作者:** <@%s>\n**发布时间:** %s\n**原帖内容:** %s",
		title,
		info.Author.ID,
		info.Timestamp,
		content,
	)
}
