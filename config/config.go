package config

import (
	_ "embed"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed default.yml
var defaultConfig []byte

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
		if !os.IsNotExist(err) {
			return nil, err
		}

		data = defaultConfig
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
