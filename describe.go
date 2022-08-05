package ecsta

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
)

type DescribeCmd struct {
	app *Ecsta

	id string
}

func NewDescribeCmd(app *Ecsta) *DescribeCmd {
	return &DescribeCmd{
		app: app,
	}
}

func (*DescribeCmd) Name() string     { return "describe" }
func (*DescribeCmd) Synopsis() string { return "describe task" }
func (*DescribeCmd) Usage() string {
	return `describe [options]:
  Describe a task in a cluster.
`
}

func (p *DescribeCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.id, "id", "", "task ID")
}

func (p *DescribeCmd) execute(ctx context.Context) error {
	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := p.app.findTask(ctx, p.id)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	f := newTaskFormatterJSON(p.app.w)
	f.AddTask(task)
	f.Close()
	return nil
}

func (p *DescribeCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
