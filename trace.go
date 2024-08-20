package ecsta

import (
	"context"
	"fmt"
	"time"

	"github.com/fujiwara/tracer"
)

type TraceOption struct {
	ID          string        `help:"task ID"`
	Duration    time.Duration `help:"duration to trace" short:"d" default:"1m"`
	SNSTopicArn string        `help:"SNS topic ARN"`
	Family      *string       `help:"task definition family name"`
	Service     *string       `help:"ECS service name"`
	JSON        bool          `help:"output JSON lines" short:"j"`
}

func (app *Ecsta) RunTrace(ctx context.Context, opt *TraceOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, &optionFindTask{id: opt.ID, family: opt.Family, service: opt.Service})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	tr, err := tracer.NewWithConfig(app.awscfg)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}

	if app.Config.Output == "json" {
		opt.JSON = true
	}
	tracerOpt := &tracer.RunOption{
		Stdout:      true,
		Duration:    opt.Duration,
		SNSTopicArn: opt.SNSTopicArn,
		JSON:        opt.JSON,
	}
	tr.SetOutput(app.w)
	return tr.Run(ctx, app.cluster, *task.TaskArn, tracerOpt)
}
