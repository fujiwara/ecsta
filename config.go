package ecsta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-jsonnet"
)

type Config struct {
	FilterCommand string `json:"filter_command"`
}

var configDir string

const configSubdir = "ecsta"

func init() {
	if h := os.Getenv("XDG_CONFIG_HOME"); h != "" {
		configDir = filepath.Join(h, configSubdir)
	} else {
		d, err := os.UserHomeDir()
		if err != nil {
			d = os.Getenv("HOME")
		}
		configDir = filepath.Join(d, ".config", configSubdir)
	}
}

func newConfig() (*Config, error) {
	vm := jsonnet.MakeVM()
	p := filepath.Join(configDir, "config")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return &Config{}, nil
	}
	jsonStr, err := vm.EvaluateFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate %s: %w", p, err)
	}
	var c Config
	if err := json.Unmarshal([]byte(jsonStr), &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", p, err)
	}
	return &c, nil
}
