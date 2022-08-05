package ecsta

import (
	"context"
	"errors"
	"io"
	"os"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

var ErrAborted = errors.New("Aborted")

type Ecsta struct {
	region string
	ecs    *ecs.Client
	w      io.Writer

	config *Config
}

func New(ctx context.Context, region string) (*Ecsta, error) {
	awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
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

type optionDescribeTasks struct {
	cluster *string
	ids     []string
}

func (app *Ecsta) describeTasks(ctx context.Context, opt *optionDescribeTasks) ([]types.Task, error) {
	out, err := app.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: opt.cluster,
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
