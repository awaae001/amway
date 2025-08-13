package model

// Config 对应于 config.yaml 的顶级结构
type Config struct {
	Token    string   `mapstructure:"TOKEN"`
	Commands Commands `mapstructure:"commands"`
	AmwayBot AmwayBot `mapstructure:"amwayBot"`
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
