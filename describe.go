package ecsta

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

type DescribeOption struct {
	ID string
}

func newDescribeCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "describe",
		Usage: "describe task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "task ID",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunDescribe(c.Context, &DescribeOption{
				ID: c.String("id"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunDescribe(ctx context.Context, opt *DescribeOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, opt.ID)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	f := newTaskFormatterJSON(app.w)
	f.AddTask(task)
	f.Close()
	return nil
}
