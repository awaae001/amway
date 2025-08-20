package utils

import (
	"amway/model"
	"encoding/base64"
	"encoding/json"
	"os"
	"time"
)

// StringPtr returns a pointer to the given string.
// This is a helper function for discordgo fields that require a *string.
func StringPtr(s string) *string {
	return &s
}

// SavePanelState 保存面板状态到JSON文件
func SavePanelState(filePath, channelID, messageID string) error {
	state := model.PanelState{
		ChannelID: channelID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadPanelState 从JSON文件加载面板状态
func LoadPanelState(filePath string) (*model.PanelState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state model.PanelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// EncodeBase64 encodes a string to a URL-safe base64 string.
func EncodeBase64(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// DecodeBase64 decodes a URL-safe base64 string.
func DecodeBase64(s string) (string, error) {
	decodedBytes, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}
