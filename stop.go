package ecsta

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/subcommands"
)

type StopCmd struct {
	app *Ecsta

	id    string
	force bool
}

func NewStopCmd(app *Ecsta) *StopCmd {
	return &StopCmd{
		app: app,
	}
}

func (*StopCmd) Name() string     { return "stop" }
func (*StopCmd) Synopsis() string { return "stop task" }
func (*StopCmd) Usage() string {
	return `stop [options]:
  Stop task in a cluster.
`
}

func (p *StopCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.id, "id", "", "task ID")
	f.BoolVar(&p.force, "force", false, "stop a task without confirmation")
}

func (p *StopCmd) execute(ctx context.Context) error {
	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := p.app.findTask(ctx, p.id)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	return p.stopTask(ctx, task)
}

func (p *StopCmd) stopTask(ctx context.Context, task types.Task) error {
	var doStop bool
	if !p.force {
		doStop = prompter.YN(fmt.Sprintf("Do you request to stop a task %s?", arnToName(*task.TaskArn)), false)
	}
	if !doStop {
		return ErrAborted
	}
	_, err := p.app.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &p.app.cluster,
		Task:    task.TaskArn,
		Reason:  aws.String("Request stop task by user action."),
	})
	if err != nil {
		return fmt.Errorf("failed to stop task %s: %w", arnToName(*task.TaskArn), err)
	}
	return nil
}

func (p *StopCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
