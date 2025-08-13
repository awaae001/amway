package config

import (
	"amway/model"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var Cfg model.Config

func LoadConfig() (err error) {
	// 首先加载.env文件
	err = godotenv.Load()
	if err != nil {
		// .env文件不存在时不报错，继续执行
	}

	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 设置环境变量前缀和自动环境变量读取
	viper.AutomaticEnv()
	viper.SetEnvPrefix("") // 不使用前缀，直接读取TOKEN环境变量

	// 显式绑定环境变量
	viper.BindEnv("token", "TOKEN")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&Cfg)
	if err != nil {
		return
	}

	// 如果配置文件中没有token，尝试从环境变量直接读取
	if Cfg.Token == "" {
		if token := os.Getenv("TOKEN"); token != "" {
			Cfg.Token = token
		}
	}

	return
}
