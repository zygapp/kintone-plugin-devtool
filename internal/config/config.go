package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ConfigDir = ".kpdev"
const ConfigFile = "config.json"

type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type DevEnvConfig struct {
	Domain string     `json:"domain"`
	Auth   AuthConfig `json:"auth,omitempty"`
}

type ProdEnvConfig struct {
	Name   string     `json:"name"`
	Domain string     `json:"domain"`
	Auth   AuthConfig `json:"auth,omitempty"`
}

type KintoneConfig struct {
	Dev  DevEnvConfig    `json:"dev"`
	Prod []ProdEnvConfig `json:"prod,omitempty"`
}

type EntryConfig struct {
	Main   string `json:"main"`
	Config string `json:"config"`
}

type DevConfig struct {
	Origin string      `json:"origin"`
	Entry  EntryConfig `json:"entry"`
}

type TargetsConfig struct {
	Desktop bool `json:"desktop"`
	Mobile  bool `json:"mobile"`
}

type Config struct {
	Kintone KintoneConfig `json:"kintone"`
	Dev     DevConfig     `json:"dev"`
	Targets TargetsConfig `json:"targets"`
}

func Load(projectDir string) (*Config, error) {
	configPath := filepath.Join(projectDir, ConfigDir, ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save(projectDir string) error {
	configDir := filepath.Join(projectDir, ConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, ConfigFile)
	return os.WriteFile(configPath, data, 0644)
}

func GetConfigDir(projectDir string) string {
	return filepath.Join(projectDir, ConfigDir)
}
