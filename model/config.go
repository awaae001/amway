package model

import "time"

// Config 对应于 config.yaml 的顶级结构
type Config struct {
	Token      string     `mapstructure:"token"`
	Debug      bool       `mapstructure:"debug"`
	Commands   Commands   `mapstructure:"commands"`
	AmwayBot   AmwayBot   `mapstructure:"amwayBot"`
	RoleConfig RoleConfig `mapstructure:"role_config"`
}

// PanelState 面板状态
type PanelState struct {
	ChannelID string    `json:"channel_id"`
	MessageID string    `json:"message_id"`
	CreatedAt time.Time `json:"created_at"`
}

// AmwayBot 对应 "amwayBot" 部分
type AmwayBot struct {
	Amway Amway `mapstructure:"amway"`
}

// Amway 对应 "amway" 部分
type Amway struct {
	ReviewChannelID  string `mapstructure:"review_channel_id"`
	PublishChannelID string `mapstructure:"publish_channel_id"`
}

// Commands 对应 "commands" 部分
type Commands struct {
	Allowguils []string `mapstructure:"allowguils"`
	Auth       Auth     `mapstructure:"auth"`
}

// Auth 对应 "auth" 部分
type Auth struct {
	Developers  []string `mapstructure:"Developers"`
	AdminsRoles []string `mapstructure:"AdminsRoles"`
	Guest       []string `mapstructure:"Guest"`
}

// RoleConfig 对应于 role_config.json 的顶级结构
type RoleConfig map[string]map[string]RoleDetail

// RoleDetail 包含每个角色的详细信息
type RoleDetail struct {
	ConfigID   int        `mapstructure:"config_id"`
	Name       string     `mapstructure:"name"`
	StartAt    string     `mapstructure:"start_at"`
	EndAt      string     `mapstructure:"end_at"`
	GRPCConfig GRPCConfig `mapstructure:"grpc_config"`
}

// GRPCConfig 包含 gRPC 服务的具体配置
type GRPCConfig struct {
	Address string `mapstructure:"address"`
	RoleID  string `mapstructure:"role_id"`
}
