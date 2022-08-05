package ecsta

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
)

type ListCmd struct {
	app *Ecsta

	family string
	output string
}

func NewListCmd(app *Ecsta) *ListCmd {
	return &ListCmd{
		app: app,
	}
}

func (*ListCmd) Name() string     { return "list" }
func (*ListCmd) Synopsis() string { return "liste tasks" }
func (*ListCmd) Usage() string {
	return `list -cluster <cluster> [options]:
  Show task ARNs in the cluster.
`
}

func (p *ListCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.family, "family", "", "task definition family")
	f.StringVar(&p.output, "output", "table", "output format (table|json|tsv)")
}

func (p *ListCmd) execute(ctx context.Context) error {
	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	tasks, err := p.app.listTasks(ctx, &optionListTasks{
		family: optional(p.family),
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster %s: %w", p.app.cluster, err)
	}
	f, err := newTaskFormatter(p.app.w, p.output, true)
	if err != nil {
		return fmt.Errorf("failed to create task formatter: %w", err)
	}
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}

func (p *ListCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
