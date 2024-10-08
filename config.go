package ecsta

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/Songmu/prompter"
)

type Config struct {
	FilterCommand   string `help:"command to run to filter messages" json:"filter_command"`
	Output          string `help:"output format (table, tsv or json)" enum:"table,tsv,json" default:"table" json:"output"`
	TaskFormatQuery string `help:"A jq query to format task in selector" json:"task_format_query"`
}

func (c *Config) ConfigElements() []ConfigElement {
	v := reflect.ValueOf(c).Elem()
	elements := make([]ConfigElement, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		elements[i] = ConfigElement{
			Name:        v.Type().Field(i).Tag.Get("json"),
			Description: v.Type().Field(i).Tag.Get("help"),
			Default:     v.Type().Field(i).Tag.Get("default"),
		}
	}
	return elements
}

func (c *Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}

func (c *Config) Get(name string) string {
	v := reflect.ValueOf(c).Elem()
	name = strings.ToLower(name)

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("json") == name {
			return v.Field(i).String()
		}
	}
	panic(fmt.Errorf("config element %s not defined", name))
}

func (c *Config) Set(name, value string) {
	v := reflect.ValueOf(c).Elem()
	name = strings.ToLower(name)

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("json") == name {
			if v.Field(i).CanSet() {
				v.Field(i).SetString(value)
				return // success
			}
		}
	}
	panic(fmt.Errorf("config element %s not defined or not settable", name))
}

func (c *Config) fillDefault() {
	v := reflect.ValueOf(c).Elem()

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).String() == "" {
			v.Field(i).SetString(v.Type().Field(i).Tag.Get("default"))
		}
	}
}

func (c *Config) Names() []string {
	v := reflect.ValueOf(c).Elem()
	var names []string

	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Tag.Get("json")
		names = append(names, name)
	}
	return names
}

func (c *Config) OverrideCLI(cli *CLI) {
	if cli.Output != "" {
		c.Output = cli.Output
	}
	if cli.TaskFormatQuery != "" {
		c.TaskFormatQuery = cli.TaskFormatQuery
	}
}

type ConfigElement struct {
	Name        string `json:"name"`
	Description string `json:"help"`
	Default     string `json:"default"`
}

var configDir string
var configOnce sync.Once

const configSubdir = "ecsta"

func setConfigDir() {
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

func newConfig() *Config {
	config := &Config{}
	config.fillDefault()
	return config
}

func configFilePath() string {
	return filepath.Join(configDir, "config.json")
}

func loadConfig() (*Config, error) {
	if config, err := loadConfigFile(); err == nil {
		config.fillDefault()
		return config, nil
	}
	return newConfig(), nil
}

func loadConfigFile() (*Config, error) {
	p := configFilePath()
	jsonStr, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	config := &Config{}
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", p, err)
	}
	return config, nil
}

func reConfigure(c *Config) error {
	slog.Info("configuration file", "path", configFilePath())
	nc := &Config{}

	for _, elm := range c.ConfigElements() {
		current := c.Get(elm.Name)
		input := prompter.Prompt(
			fmt.Sprintf("Enter %s (%s)", elm.Name, elm.Description),
			current,
		)
		nc.Set(elm.Name, input)
	}
	return saveConfig(nc)
}

func saveConfig(c *Config) error {
	p := configFilePath()
	if _, err := os.Stat(configDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return fmt.Errorf("failed to stat config directory: %w", err)
		}
	}
	b, err := json.MarshalIndent(c, "", "  ")
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
	slog.Info("saved configuration file", "path", configFilePath())
	return nil
}
