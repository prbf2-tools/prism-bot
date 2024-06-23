package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type PRISM struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"user"`
	Password string `yaml:"pass"`
}

type Discord struct {
	Token   string `yaml:"token"`
	AppID   string `yaml:"appID"`
	GuildID string `yaml:"guildID"`
}

type Channel struct {
	ID       string `yaml:"id"`
	Template string `yaml:"template"`
}

type ServerDetails struct {
	Channels []Channel `yaml:"channels"`
}

type Config struct {
	PRISM         PRISM         `yaml:"prism"`
	Discord       Discord       `yaml:"discord"`
	ServerDetails ServerDetails `yaml:"serverDetails"`
}

func NewConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
