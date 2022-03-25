package ecsta

import (
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

func (p *TasksCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	args := f.Args()
	if len(args) > 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	err := func() error {
		switch len(args) {
		case 0:
			clusters, err := p.app.listClusters(ctx)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}
			for _, cluster := range clusters {
				fmt.Fprintln(p.app.w, cluster)
			}
		case 1:
			cluster := args[0]
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
		default:
			panic("invalid number of arguments")
		}
		return nil
	}()
	if err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
