package ecsta

import (
	"context"
	"fmt"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/urfave/cli/v2"
)

type StopOption struct {
	ID    string
	Force bool
}

func newStopCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "stop",
		Usage: "stop task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "task ID",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "stop without confirmation",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunStop(c.Context, &StopOption{
				ID:    c.String("id"),
				Force: c.Bool("force"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunStop(ctx context.Context, opt *StopOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, opt.ID)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	var doStop bool
	if !opt.Force {
		doStop = prompter.YN(fmt.Sprintf("Do you request to stop a task %s?", arnToName(*task.TaskArn)), false)
	}
	if !doStop {
		return ErrAborted
	}
	if _, err := app.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &app.cluster,
		Task:    task.TaskArn,
		Reason:  aws.String("Request stop task by user action."),
	}); err != nil {
		return fmt.Errorf("failed to stop task %s: %w", arnToName(*task.TaskArn), err)
	}
	return nil
}
