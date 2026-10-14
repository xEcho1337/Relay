package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type NetworkConfig struct {
	Port         int  `yaml:"port"`
	IPForwarding bool `yaml:"ip-forwarding"`
	OnlineMode   bool `yaml:"online-mode"`
}

type ServerConfig struct {
	ID      string `yaml:"id"`
	Address string `yaml:"address"`
}

type Config struct {
	Network NetworkConfig           `yaml:"network"`
	Servers map[string]ServerConfig `yaml:"servers"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
