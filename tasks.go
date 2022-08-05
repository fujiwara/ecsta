package ecsta

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/subcommands"
)

type TasksCmd struct {
	app *Ecsta

	cluster string
	family  string
	output  string
	id      string
	find    bool
	stop    bool
	force   bool
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
	f.BoolVar(&p.find, "find", false, "find a task")
	f.BoolVar(&p.stop, "stop", false, "stop a task")
	f.BoolVar(&p.force, "force", false, "stop a task without confirmation")
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
	switch {
	case p.find:
		tasks, err = p.selectTasks(ctx, tasks)
		if err != nil {
			return fmt.Errorf("failed to select tasks: %w", err)
		}
		p.output = "json"
	case p.stop:
		tasks, err := p.selectTasks(ctx, tasks)
		if err != nil {
			return fmt.Errorf("failed to select tasks: %w", err)
		}
		return p.stopTask(ctx, tasks[0])
	default:
	}
	f, _ := newTaskFormatter(p.app.w, p.output, true)
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}

func (p *TasksCmd) stopTask(ctx context.Context, task types.Task) error {
	var doStop bool
	if !p.force {
		doStop = prompter.YN(fmt.Sprintf("Do you request to stop a task %s?", arnToName(*task.TaskArn)), false)
	}
	if !doStop {
		return ErrAborted
	}
	_, err := p.app.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &p.cluster,
		Task:    task.TaskArn,
		Reason:  aws.String("Request stop task by user action."),
	})
	if err != nil {
		return fmt.Errorf("failed to stop task %s: %w", arnToName(*task.TaskArn), err)
	}
	return nil
}

func (p *TasksCmd) selectTasks(ctx context.Context, tasks []types.Task) ([]types.Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found")
	}
	buf := new(bytes.Buffer)
	f, _ := newTaskFormatter(buf, "tsv", false)
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	res, err := p.app.runFilter(buf, "")
	if err != nil {
		return nil, fmt.Errorf("failed to run filter: %w", err)
	}
	p.find = false
	id := strings.SplitN(res, "\t", 2)[0]
	for _, task := range tasks {
		if arnToName(*task.TaskArn) == id {
			return []types.Task{task}, nil
		}
	}
	return nil, fmt.Errorf("task %s not found", id)
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
