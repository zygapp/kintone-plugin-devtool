package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type EnvConfig struct {
	Username string
	Password string
}

func LoadEnv(projectDir string) (*EnvConfig, error) {
	envPath := filepath.Join(projectDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			return nil, err
		}
	}

	cfg := &EnvConfig{
		Username: os.Getenv("KPDEV_USERNAME"),
		Password: os.Getenv("KPDEV_PASSWORD"),
	}

	return cfg, nil
}

func (c *EnvConfig) HasAuth() bool {
	return c.Username != "" && c.Password != ""
}
