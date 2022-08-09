package ecsta

import (
	"context"
	"fmt"
	"time"

	"github.com/fujiwara/tracer"
	"github.com/urfave/cli/v2"
)

type TraceOption struct {
	ID          string
	Duration    time.Duration
	SNSTopicArn string
}

func newTraceCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "trace",
		Usage: "trace task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "task ID",
			},
			&cli.DurationFlag{
				Name:  "duration",
				Usage: "duration to trace",
				Value: time.Minute,
			},
			&cli.StringFlag{
				Name:  "sns-topic-arn",
				Usage: "SNS topic ARN",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunTrace(c.Context, &TraceOption{
				ID:          c.String("id"),
				Duration:    c.Duration("duration"),
				SNSTopicArn: c.String("sns-topic-arn"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunTrace(ctx context.Context, opt *TraceOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, opt.ID)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	tr, err := tracer.NewWithConfig(app.awscfg)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}
	tracerOpt := &tracer.RunOption{
		Stdout:      true,
		Duration:    opt.Duration,
		SNSTopicArn: opt.SNSTopicArn,
	}
	return tr.Run(ctx, app.cluster, *task.TaskArn, tracerOpt)
}
