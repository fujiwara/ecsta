package ecsta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

var ErrAborted = errors.New("Aborted")

type Ecsta struct {
	region  string
	cluster string

	ecs *ecs.Client
	w   io.Writer

	config *Config
}

func New(ctx context.Context, region, cluster string) (*Ecsta, error) {
	awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	conf, err := newConfig()
	if err != nil {
		return nil, err
	}
	app := &Ecsta{
		cluster: cluster,
		region:  awscfg.Region,
		ecs:     ecs.NewFromConfig(awscfg),
		w:       os.Stdout,
		config:  conf,
	}
	return app, nil
}

type optionListTasks struct {
	family  *string
	service *string
}

type optionDescribeTasks struct {
	ids []string
}

func (app *Ecsta) describeTasks(ctx context.Context, opt *optionDescribeTasks) ([]types.Task, error) {
	out, err := app.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &app.cluster,
		Tasks:   opt.ids,
		Include: []types.TaskField{"TAGS"},
	})
	if err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

func (app *Ecsta) listTasks(ctx context.Context, opt *optionListTasks) ([]types.Task, error) {
	tasks := []types.Task{}
	tp := ecs.NewListTasksPaginator(
		app.ecs,
		&ecs.ListTasksInput{
			Cluster:     &app.cluster,
			Family:      opt.family,
			ServiceName: opt.service,
		},
	)
	for tp.HasMorePages() {
		to, err := tp.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		if len(to.TaskArns) == 0 {
			continue
		}
		out, err := app.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &app.cluster,
			Tasks:   to.TaskArns,
			Include: []types.TaskField{"TAGS"},
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, out.Tasks...)
	}
	return tasks, nil
}

func (app *Ecsta) listClusters(ctx context.Context) ([]string, error) {
	clusters := []string{}
	tp := ecs.NewListClustersPaginator(app.ecs, &ecs.ListClustersInput{})
	for tp.HasMorePages() {
		out, err := tp.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		if len(out.ClusterArns) == 0 {
			break
		}
		clusters = append(clusters, out.ClusterArns...)
	}
	return clusters, nil
}

func (app *Ecsta) selectCluster(ctx context.Context) (string, error) {
	clusters, err := app.listClusters(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	buf := new(bytes.Buffer)
	for _, cluster := range clusters {
		fmt.Fprintln(buf, arnToName(cluster))
	}
	res, err := app.runFilter(buf, "")
	if err != nil {
		return "", fmt.Errorf("failed to run filter: %w", err)
	}
	return res, nil
}

func (app *Ecsta) findTask(ctx context.Context, id string) (types.Task, error) {
	if id != "" {
		tasks, err := app.describeTasks(ctx, &optionDescribeTasks{ids: []string{id}})
		if err != nil {
			return types.Task{}, err
		}
		if len(tasks) == 1 {
			return tasks[0], nil
		}
	}
	tasks, err := app.listTasks(ctx, &optionListTasks{})
	if err != nil {
		return types.Task{}, err
	}
	buf := new(bytes.Buffer)
	f, _ := newTaskFormatter(buf, "tsv", false)
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	res, err := app.runFilter(buf, "")
	if err != nil {
		return types.Task{}, fmt.Errorf("failed to run filter: %w", err)
	}
	id = strings.SplitN(res, "\t", 2)[0]
	for _, task := range tasks {
		if arnToName(*task.TaskArn) == id {
			return task, nil
		}
	}
	return types.Task{}, fmt.Errorf("task %s not found", id)
}

func (app *Ecsta) SetCluster(ctx context.Context) error {
	if app.cluster == "" {
		cluster, err := app.selectCluster(ctx)
		if err != nil {
			return err
		}
		app.cluster = cluster
	}
	return nil
}
