package ecsta

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Cluster string `help:"ECS cluster name" short:"c" env:"ECS_CLUSTER"`
	Region  string `help:"AWS region" short:"r" env:"AWS_REGION"`
	Output  string `help:"output format (table, tsv, json)" short:"o" default:"table" enum:"table,tsv,json"`

	Configure *ConfigureOption `cmd:"" help:"Create a configuration file of ecsta"`
}

func RunCLI(ctx context.Context, args []string) error {
	var cli CLI
	parser, err := kong.New(&cli, kong.Vars{"version": Version})
	if err != nil {
		return err
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	app, err := New(ctx, cli.Region, cli.Cluster)
	if err != nil {
		return err
	}
	// app.Config.OverrideCLI()
	switch strings.Fields(kctx.Command())[0] {
	case "configure":
		return app.RunConfigure(ctx, cli.Configure)
	}
	return fmt.Errorf("unknown command: %s", kctx.Command())
}

/*
func (config Config) OverrideCLI(c *cli.Context) {
	for _, elm := range ConfigElements {
		if v := c.String(elm.Name); v != "" {
			config.Set(elm.Name, v)
		}
	}
}
*/
