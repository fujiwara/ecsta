package ecsta

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
)

type TasksCmd struct {
	app *Ecsta

	family string
	output string
}

func NewTasksCmd(app *Ecsta) *TasksCmd {
	return &TasksCmd{
		app: app,
	}
}

func (*TasksCmd) Name() string     { return "tasks" }
func (*TasksCmd) Synopsis() string { return "manage tasks" }
func (*TasksCmd) Usage() string {
	return `tasks <cluster>:
  Show task ARNs in the cluster.
`
}

func (p *TasksCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.family, "family", "", "task definition family")
	f.StringVar(&p.output, "output", "table", "output format (table|json|tsv)")
}

func (p *TasksCmd) selectCluster(ctx context.Context) (string, error) {
	clusters, err := p.app.listClusters(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	buf := new(bytes.Buffer)
	for _, cluster := range clusters {
		fmt.Fprintln(buf, arnToName(cluster))
	}
	res, err := p.app.runFilter(buf, p.output)
	if err != nil {
		return "", fmt.Errorf("failed to run filter: %w", err)
	}
	return res, nil
}

func (p *TasksCmd) execute(ctx context.Context, cluster string) error {
	tasks, err := p.app.listTasks(ctx, &optionListTasks{
		cluster: &cluster,
		family:  optional(p.family),
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster %s: %w", cluster, err)
	}
	f, _ := newTaskFormatter(p.output, p.app.w)
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}

func (p *TasksCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	args := f.Args()
	switch len(args) {
	case 0:
		cluster, err := p.selectCluster(ctx)
		if err != nil {
			log.Println("[error]", err)
			return subcommands.ExitFailure
		}
		if err := p.execute(ctx, cluster); err != nil {
			log.Println("[error]", err)
			return subcommands.ExitFailure
		}
	case 1:
		if err := p.execute(ctx, args[0]); err != nil {
			log.Println("[error]", err)
			return subcommands.ExitFailure
		}
	default:
		return subcommands.ExitUsageError
	}
	return subcommands.ExitFailure
}
