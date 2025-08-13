package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Token string `mapstructure:"TOKEN"`
}

var Cfg Config

func LoadConfig() (err error) {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&Cfg)
	return
}
