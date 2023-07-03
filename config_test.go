package ecsta_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/fujiwara/ecsta"
	"github.com/google/go-cmp/cmp"
)

func TestConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "testdata/config")
	ecsta.SetConfigDir()

	if err := os.WriteFile("testdata/config/ecsta/config.json", []byte(`{
	"filter_command": "FILTER_COMMAND",
	"output": "OUTPUT",
	"task_format_query": "TASK_FORMAT_QUERY"
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

	for _, key := range []string{"filter_command", "output", "task_format_query"} {
		if conf.Get(key) != strings.ToUpper(key) {
			t.Errorf("unexpected config %s value: %s", key, conf.Get(key))
		}
		conf.Set(key, fmt.Sprintf("value of %s", key))
	}
	if err := ecsta.SaveConfig(conf); err != nil {
		t.Fatal(err)
	}

	conf, err = ecsta.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"filter_command", "output", "task_format_query"} {
		if conf.Get(key) != fmt.Sprintf("value of %s", key) {
			t.Errorf("unexpected config %s value: %s", key, conf.Get(key))
		}
	}
}
