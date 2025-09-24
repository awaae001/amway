package config

import (
	"amway/model"
	"fmt"
	"log"
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

	// 设置环境变量前缀和自动环境变量读取
	viper.AutomaticEnv()
	viper.SetEnvPrefix("") // 不使用前缀，直接读取TOKEN环境变量
	// 显式绑定环境变量
	viper.BindEnv("token", "TOKEN")

	// 1. 读取主配置文件 (config.yaml)
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在也没关系，可能全部使用环境变量
			log.Printf("未找到主配置文件 (config.yaml)，将依赖环境变量。")
		} else {
			// 配置文件存在但解析错误
			return fmt.Errorf("读取主配置文件时发生错误: %w", err)
		}
	}

	// 2. 合并角色配置文件 (role_config.json)
	viper.SetConfigName("role_config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到角色配置文件 (config/role_config.json)，将跳过合并。")
		} else {
			// 合并时发生其他错误
			return fmt.Errorf("合并角色配置文件时发生错误: %w", err)
		}
	}

	// 3. Unmarshal所有配置到结构体
	err = viper.Unmarshal(&Cfg)
	if err != nil {
		return fmt.Errorf("解析配置到结构体时发生错误: %w", err)
	}

	// 4. 环境变量回退 (如果需要)
	// viper.AutomaticEnv() 已经处理了大部分情况
	// 这里的代码是为了双重保证
	if Cfg.Token == "" {
		if token := os.Getenv("TOKEN"); token != "" {
			Cfg.Token = token
		}
	}

	return nil
}
