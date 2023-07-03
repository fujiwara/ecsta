package ecsta

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Songmu/prompter"
)

type Config map[string]string

func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}

func (c Config) Get(name string) string {
	for _, elm := range ConfigElements {
		if elm.Name == name {
			return c[name]
		}
	}
	panic(fmt.Errorf("config element %s not defined", name))
}

func (c Config) Set(name, value string) {
	for _, elm := range ConfigElements {
		if elm.Name == name {
			c[name] = value
			return
		}
	}
	panic(fmt.Errorf("config element %s not defined", name))
}

type ConfigElement struct {
	Name        string `json:"name"`
	Description string `json:"help"`
	Default     string `json:"default"`
}

var ConfigElements = []ConfigElement{
	{
		Name:        "filter_command",
		Description: "command to run to filter messages",
		Default:     "",
	},
	{
		Name:        "output",
		Description: "output format (table, tsv or json)",
		Default:     "table",
	},
	{
		Name:        "task_format_query",
		Description: "A jq query to format task in selector",
		Default:     "",
	},
}

var xdg = struct {
	ConfigDir string
	StateDir  string
	Subdir    string
}{
	Subdir: "ecsta",
}

func init() {
	if h := os.Getenv("XDG_CONFIG_HOME"); h != "" {
		xdg.ConfigDir = filepath.Join(h, xdg.Subdir)
	} else {
		d, err := os.UserHomeDir()
		if err != nil {
			d = os.Getenv("HOME")
		}
		xdg.ConfigDir = filepath.Join(d, ".config", xdg.Subdir)
	}
	if h := os.Getenv("XDG_STATE_HOME"); h != "" {
		xdg.StateDir = filepath.Join(h, xdg.Subdir)
	} else {
		d, err := os.UserHomeDir()
		if err != nil {
			d = os.Getenv("HOME")
		}
		xdg.StateDir = filepath.Join(d, ".local", "state", xdg.Subdir)
	}
}

func newConfig() Config {
	config := Config{}
	config.fillDefault()
	return config
}

func configFilePath() string {
	return filepath.Join(xdg.ConfigDir, "config.json")
}

func loadConfig() (Config, error) {
	if config, err := loadConfigFile(); err == nil {
		config.fillDefault()
		return config, nil
	}
	return newConfig(), nil
}

func (config Config) fillDefault() {
	for _, elm := range ConfigElements {
		if config[elm.Name] == "" && elm.Default != "" {
			config[elm.Name] = elm.Default
		}
	}
}

func loadConfigFile() (Config, error) {
	p := configFilePath()
	jsonStr, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	var config Config
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", p, err)
	}
	return config, nil
}

func reConfigure(config Config) error {
	log.Println("configuration file:", configFilePath())
	newConfig := Config{}

	for _, elm := range ConfigElements {
		current := config.Get(elm.Name)
		input := prompter.Prompt(
			fmt.Sprintf("Enter %s (%s)", elm.Name, elm.Description),
			current,
		)
		newConfig.Set(elm.Name, input)
	}
	return saveConfig(newConfig)
}

func saveConfig(config Config) error {
	p := configFilePath()
	if _, err := os.Stat(xdg.ConfigDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(xdg.ConfigDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return fmt.Errorf("failed to stat config directory: %w", err)
		}
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if _, err := os.Stat(p); err == nil {
		if err := os.Rename(p, p+".bak"); err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
	}
	if err := os.WriteFile(p, b, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	log.Println("Saved configuration file:", configFilePath())
	return nil
}

func prepareConsoleHistory() error {
	if _, err := os.Stat(xdg.StateDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(xdg.StateDir, 0711); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return fmt.Errorf("failed to stat config directory: %w", err)
		}
	}
	return nil
}
