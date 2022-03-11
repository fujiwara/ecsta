package ecsta

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type TasksCmd struct {
	app *Ecsta

	capitalize bool
}

func NewTasksCmd(app *Ecsta) *TasksCmd {
	return &TasksCmd{
		app: app,
	}
}

func (*TasksCmd) Name() string     { return "tasks" }
func (*TasksCmd) Synopsis() string { return "list tasks" }
func (*TasksCmd) Usage() string {
	return `tasks <cluster>:
  Show task ARNs in the cluster.
`
}

func (p *TasksCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.capitalize, "capitalize", false, "capitalize output")
}

func (p *TasksCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	args := f.Args()
	switch len(args) {
	case 0:
		clusters, err := p.app.listClusters()
		if err != nil {
			panic(err)
		}
		for _, cluster := range clusters {
			fmt.Println(cluster)
		}
	case 1:
		tasks, err := p.app.listTasks(args[0])
		if err != nil {
			panic(err)
		}
		for _, task := range tasks {
			fmt.Println(*task.TaskArn)
		}
	default:
		f.Usage()
	}
	return subcommands.ExitSuccess
}
