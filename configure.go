package ecsta

import (
	"context"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"
)

type ConfigureOption struct {
	Show bool
}

func newConfigureCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "configure",
		Usage: "Create a configuration file of ecsta",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "show",
				Aliases: []string{"s"},
				Usage:   "show a current configuration",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunConfigure(c.Context, &ConfigureOption{
				Show: c.Bool("show"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunConfigure(ctx context.Context, opt *ConfigureOption) error {
	if opt.Show {
		log.Println("configuration file:", configFilePath())
		fmt.Fprintln(app.w, app.Config.String())
		return nil
	}
	if err := reConfigure(app.Config); err != nil {
		return err
	}
	return nil
}
