package ecsta

import (
	"context"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Ecsta struct {
	region string
	ecs    *ecs.Client
	w      io.Writer

	config *Config
}

func New(ctx context.Context, region string) (*Ecsta, error) {
	awscfg, err := awsConfig.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	conf, err := newConfig()
	if err != nil {
		return nil, err
	}
	return &Ecsta{
		region: awscfg.Region,
		ecs:    ecs.NewFromConfig(awscfg),
		w:      os.Stdout,
		config: conf,
	}, nil
}

type optionListTasks struct {
	cluster *string
	family  *string
	service *string
}

func (app *Ecsta) listTasks(ctx context.Context, opt *optionListTasks) ([]types.Task, error) {
	tasks := []types.Task{}
	tp := ecs.NewListTasksPaginator(
		app.ecs,
		&ecs.ListTasksInput{
			Cluster:     opt.cluster,
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
			Cluster: opt.cluster,
			Tasks:   to.TaskArns,
			Include: []types.TaskField{"TAGS"},
		})
		if err != nil {
			return nil, err
		}
		for _, task := range out.Tasks {
			tasks = append(tasks, task)
		}
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
