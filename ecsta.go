package ecsta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/samber/lo"
)

var ErrAborted = errors.New("Aborted")
var Version string

type Ecsta struct {
	Config Config

	region  string
	cluster string

	awscfg aws.Config
	ecs    *ecs.Client
	ssm    *ssm.Client
	logs   *cloudwatchlogs.Client
	w      io.Writer
}

func New(ctx context.Context, region, cluster string) (*Ecsta, error) {
	awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	conf, err := loadConfig()
	if err != nil {
		return nil, err
	}
	app := &Ecsta{
		Config: conf,

		cluster: cluster,
		region:  awscfg.Region,
		awscfg:  awscfg,
		ecs:     ecs.NewFromConfig(awscfg),
		ssm:     ssm.NewFromConfig(awscfg),
		logs:    cloudwatchlogs.NewFromConfig(awscfg),
		w:       os.Stdout,
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
	var inputs []*ecs.ListTasksInput
	if opt.family != nil && opt.service != nil {
		// LisTasks does not support family and service at the same time
		// spilit into two requests
		inputs = []*ecs.ListTasksInput{
			{Cluster: &app.cluster, Family: opt.family},
			{Cluster: &app.cluster, ServiceName: opt.service},
		}
	} else {
		inputs = []*ecs.ListTasksInput{
			{Cluster: &app.cluster, Family: opt.family, ServiceName: opt.service},
		}
	}
	for _, input := range inputs {
		for _, status := range []types.DesiredStatus{types.DesiredStatusRunning, types.DesiredStatusStopped} {
			input := input
			input.DesiredStatus = status
			tp := ecs.NewListTasksPaginator(app.ecs, input)
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
		}
	}
	return lo.UniqBy(tasks, func(task types.Task) string {
		return aws.ToString(task.TaskArn)
	}), nil
}

func (app *Ecsta) getCluster(ctx context.Context, name string) (*types.Cluster, error) {
	out, err := app.ecs.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{name},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Clusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", name)
	}
	return &out.Clusters[0], nil
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
	res, err := app.runFilter(buf, "cluster name")
	if err != nil {
		return "", fmt.Errorf("failed to run filter: %w", err)
	}
	return res, nil
}

func (app *Ecsta) selectByFilter(ctx context.Context, src []string, title string) (string, error) {
	buf := new(bytes.Buffer)
	for _, s := range src {
		fmt.Fprintln(buf, s)
	}
	res, err := app.runFilter(buf, title)
	if err != nil {
		return "", fmt.Errorf("failed to run filter: %w", err)
	}
	return res, nil
}

var selectFuncExcludeStopped = func(task types.Task) bool {
	return aws.ToString(task.LastStatus) != "STOPPED"
}

type optionFindTask struct {
	id         string
	family     *string
	service    *string
	selectFunc func(types.Task) bool
}

func (app *Ecsta) findTask(ctx context.Context, opt *optionFindTask) (types.Task, error) {
	if opt.id != "" {
		tasks, err := app.describeTasks(ctx, &optionDescribeTasks{ids: []string{opt.id}})
		if err != nil {
			return types.Task{}, err
		}
		if len(tasks) == 1 {
			return tasks[0], nil
		}
	}
	tasks, err := app.listTasks(ctx, &optionListTasks{
		family:  opt.family,
		service: opt.service,
	})
	if err != nil {
		return types.Task{}, err
	}
	if opt.selectFunc != nil {
		tasks = lo.Filter(tasks, func(task types.Task, _ int) bool {
			return opt.selectFunc(task)
		})
	}

	buf := new(bytes.Buffer)
	fopt := formatterOption{
		Format:       "tsv",
		HasHeader:    false,
		AppendTaskID: true,
	} // default
	if query := app.Config.Get("task_format_query"); query != "" {
		fopt.Format = "json"
		fopt.Query = query
	}
	f, err := newTaskFormatter(buf, fopt)
	if err != nil {
		return types.Task{}, fmt.Errorf("failed to create formatter: %w", err)
	}
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	res, err := app.runFilter(buf, "task ID")
	if err != nil {
		return types.Task{}, fmt.Errorf("failed to run filter: %w", err)
	}
	id := strings.SplitN(res, "\t", 2)[0]
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

func (app *Ecsta) Endpoint(ctx context.Context) (string, error) {
	out, err := app.ecs.DiscoverPollEndpoint(ctx, &ecs.DiscoverPollEndpointInput{
		Cluster: &app.cluster,
	})
	if err != nil {
		return "", fmt.Errorf("failed to discover poll endpoint: %w", err)
	}
	return *out.Endpoint, nil
}

func (app *Ecsta) findContainerName(ctx context.Context, task types.Task, name string) (string, error) {
	if name != "" {
		return name, nil
	}
	if len(task.Containers) == 1 {
		return *task.Containers[0].Name, nil
	}
	containerNames := make([]string, 0, len(task.Containers))
	for _, container := range task.Containers {
		containerNames = append(containerNames, *container.Name)
	}
	container, err := app.selectByFilter(ctx, containerNames, "container name")
	if err != nil {
		return "", err
	}
	return container, nil
}
