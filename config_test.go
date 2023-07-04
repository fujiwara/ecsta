package ecsta_test

import (
	"os"
	"testing"

	"github.com/fujiwara/ecsta"
	"github.com/google/go-cmp/cmp"
)

func TestConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "testdata/config")
	ecsta.SetConfigDir()

	if err := os.WriteFile("testdata/config/ecsta/config.json", []byte(`{
	"filter_command": "peco",
	"output": "",
	"task_format_query": ".id"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	conf, err := ecsta.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	names := conf.Names()
	if d := cmp.Diff(names, []string{"filter_command", "output", "task_format_query"}); d != "" {
		t.Errorf("unexpected config names: %s", d)
	}

	if conf.Get("filter_command") != "peco" {
		t.Errorf("unexpected config filter_command value: %s", conf.Get("filter_command"))
	}
	if conf.Get("output") != "table" { // defualt value
		t.Errorf("unexpected config output value: %s", conf.Get("output"))
	}
	if conf.Get("task_format_query") != ".id" {
		t.Errorf("unexpected config task_format_query value: %s", conf.Get("task_format_query"))
	}

	conf.Set("filter_command", "fzf")
	conf.Set("output", "tsv")
	conf.Set("task_format_query", ".name")

	if err := ecsta.SaveConfig(conf); err != nil {
		t.Fatal(err)
	}

	conf, err = ecsta.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if conf.Get("filter_command") != "fzf" {
		t.Errorf("unexpected config filter_command value: %s", conf.Get("filter_command"))
	}
	if conf.Get("output") != "tsv" {
		t.Errorf("unexpected config output value: %s", conf.Get("output"))
	}
	if conf.Get("task_format_query") != ".name" {
		t.Errorf("unexpected config task_format_query value: %s", conf.Get("task_format_query"))
	}
}
