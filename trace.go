package ecsta

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/fujiwara/tracer"
	"github.com/google/subcommands"
)

type TraceCmd struct {
	app *Ecsta

	id          string
	duration    time.Duration
	snsTopicArn string
}

func NewTraceCmd(app *Ecsta) *TraceCmd {
	return &TraceCmd{
		app: app,
	}
}

func (*TraceCmd) Name() string     { return "trace" }
func (*TraceCmd) Synopsis() string { return "trace task" }
func (*TraceCmd) Usage() string {
	return `trace [options]:
  Trace task in a cluster.
`
}

func (p *TraceCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.id, "id", "", "task ID")
	f.DurationVar(&p.duration, "duration", time.Minute*5, "duration")
	f.StringVar(&p.snsTopicArn, "sns-topic-arn", "", "SNS topic ARN")
}

func (p *TraceCmd) execute(ctx context.Context) error {
	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := p.app.findTask(ctx, p.id)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	return p.traceTask(ctx, task)
}

func (p *TraceCmd) traceTask(ctx context.Context, task types.Task) error {
	tr, err := tracer.NewWithConfig(p.app.awscfg)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}
	opt := &tracer.RunOption{
		Stdout:      true,
		Duration:    p.duration,
		SNSTopicArn: p.snsTopicArn,
	}
	return tr.Run(ctx, p.app.cluster, *task.TaskArn, opt)
}

func (p *TraceCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
