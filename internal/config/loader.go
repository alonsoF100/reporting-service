package config

import (
	"log"

	"github.com/spf13/viper"
)

func Load() *Config {
	viper.SetConfigFile("config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Failed to read config file err:", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal("Failed to decode config err:", err)
	}

	log.Println("Config loaded successfully")
	return &config
}
