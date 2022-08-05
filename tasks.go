package ecsta

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/subcommands"
)

type TasksCmd struct {
	app *Ecsta

	cluster string
	family  string
	output  string
	id      string
}

func NewTasksCmd(app *Ecsta) *TasksCmd {
	return &TasksCmd{
		app: app,
	}
}

func (*TasksCmd) Name() string     { return "tasks" }
func (*TasksCmd) Synopsis() string { return "manage tasks" }
func (*TasksCmd) Usage() string {
	return `tasks -cluster <cluster> [options]:
  Show task ARNs in the cluster.
`
}

func (p *TasksCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.cluster, "cluster", "", "ECS cluster name")
	f.StringVar(&p.family, "family", "", "task definition family")
	f.StringVar(&p.output, "output", "table", "output format (table|json|tsv)")
	f.StringVar(&p.id, "id", "", "task ID")
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

func (p *TasksCmd) execute(ctx context.Context) error {
	var tasks []types.Task
	var err error
	if p.id != "" {
		tasks, err = p.app.describeTasks(ctx, &optionDescribeTasks{
			cluster: &p.cluster,
			ids:     []string{p.id},
		})
	} else {
		tasks, err = p.app.listTasks(ctx, &optionListTasks{
			cluster: &p.cluster,
			family:  optional(p.family),
		})
	}
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster %s: %w", p.cluster, err)
	}
	f, _ := newTaskFormatter(p.output, p.app.w)
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}

func (p *TasksCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.cluster == "" {
		cluster, err := p.selectCluster(ctx)
		if err != nil {
			log.Println("[error]", err)
			return subcommands.ExitFailure
		}
		p.cluster = cluster
	}
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
