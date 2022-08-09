package ecsta

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

type ListOption struct {
	Family  string
	Service string
}

func newListCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List tasks",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "family",
				Aliases: []string{"f"},
				Usage:   "Task definition family",
			},
			&cli.StringFlag{
				Name:    "service",
				Aliases: []string{"s"},
				Usage:   "Service name",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunList(c.Context, &ListOption{
				Family:  c.String("family"),
				Service: c.String("service"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunList(ctx context.Context, opt *ListOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	tasks, err := app.listTasks(ctx, &optionListTasks{
		family:  optional(opt.Family),
		service: optional(opt.Service),
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster %s: %w", app.cluster, err)
	}
	f, err := newTaskFormatter(app.w, app.config.Get("output"), true)
	if err != nil {
		return fmt.Errorf("failed to create task formatter: %w", err)
	}
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}
