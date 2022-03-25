package ecsta

import (
	"os"
	"path/filepath"

	"github.com/kayac/go-config"
)

type Config struct {
	FilterCommand string `yaml:"filter_command"`
}

func newConfig() (*Config, error) {
	d, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(d, ".config/ecsta/config")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return &Config{}, nil
	}
	var c Config
	if err := config.Load(&c, p); err != nil {
		return nil, err
	}
	return &c, nil
}
