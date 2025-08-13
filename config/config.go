package config

import (
	"amway/model"

	"github.com/spf13/viper"
)

var Cfg model.Config

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
