package ecsta

import (
	"github.com/urfave/cli/v2"
)

var globalFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "cluster",
		Aliases: []string{"c"},
		Usage:   "ECS cluster name",
		EnvVars: []string{"ECS_CLUSTER"},
	},
	&cli.StringFlag{
		Name:    "region",
		Aliases: []string{"r"},
		Usage:   "AWS region",
		EnvVars: []string{"AWS_REGION"},
	},
	&cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "Output format (table, tsv, json)",
		Value:   "",
	},
}

func NewCLI() *cli.App {
	return &cli.App{
		Name:           "ecsta",
		Usage:          "ECS task assistant",
		Flags:          globalFlags,
		DefaultCommand: "help",
		Commands: []*cli.Command{
			newConfigureCommand(),
			newDescribeCommand(),
			newExecCommand(),
			newListCommand(),
			newLogsCommand(),
			newPortforwardCommand(),
			newStopCommand(),
			newTraceCommand(),
		},
	}
}

func NewFromCLI(c *cli.Context) (*Ecsta, error) {
	app, err := New(c.Context, c.String("region"), c.String("cluster"))
	if err != nil {
		return nil, err
	}
	app.config.OverrideCLI(c)
	return app, nil
}

func (config Config) OverrideCLI(c *cli.Context) {
	for _, elm := range ConfigElements {
		if v := c.String(elm.Name); v != "" {
			config.Set(elm.Name, v)
		}
	}
}
