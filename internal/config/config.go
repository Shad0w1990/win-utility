package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name     string `yaml:"name"`
		Version  string `yaml:"version"`
		Language string `yaml:"language"` // "en" or "fa"
	} `yaml:"app"`
	Modules struct {
		Hogs   bool `yaml:"hogs"`
		Troll  bool `yaml:"troll"`
		Input  bool `yaml:"input"`
		AI     bool `yaml:"ai"`
	} `yaml:"modules"`
}

var GlobalConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &GlobalConfig)
}

func DefaultConfig() Config {
	c := Config{}
	c.App.Name = "Fart Alert (SysGuard)"
	c.App.Version = "0.1.0"
	c.App.Language = "en"
	c.Modules.Hogs = true
	c.Modules.Troll = true
	c.Modules.Input = true
	c.Modules.AI = false
	return c
}
