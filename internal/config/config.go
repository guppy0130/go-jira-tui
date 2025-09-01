package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type LoggerFormat string

const (
	LoggerFormatJSON LoggerFormat = "json"
	LoggerFormatText LoggerFormat = "text"
)

// Config for the app
type Config struct {
	Email     string       `mapstructurue:"email"`    // your email to sign into atlassian
	Token     string       `mapstructure:"token"`     // a token for basic auth (tested only with cloud)
	Url       string       `mapstructure:"url"`       // root URL; e.g., https://guppy0130.atlassian.net
	LogFormat LoggerFormat `mapstructure:"logformat"` // json or text
}

func LoadViper() Config {
	var config Config

	viper.SetDefault("LogFormat", LoggerFormatJSON)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if config_dir, err := os.UserConfigDir(); err != nil {
		viper.AddConfigPath(filepath.Join(config_dir, "go-jira-tui"))
	}
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("no config file? %w", err))
	}
	err := viper.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("bad config file? %w", err))
	}

	// validate?
	if config.Email == "" || config.Token == "" || config.Url == "" {
		panic(fmt.Errorf("part of config is empty: %+v", config))
	}

	return config
}
