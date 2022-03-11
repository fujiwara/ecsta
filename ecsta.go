package ecsta

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Ecsta struct {
	region string
	ctx    context.Context
	ecs    *ecs.Client
}

func New(ctx context.Context, region string) (*Ecsta, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &Ecsta{
		region: cfg.Region,
		ctx:    ctx,
		ecs:    ecs.NewFromConfig(cfg),
	}, nil
}

func (app *Ecsta) listTasks(cluster string) ([]types.Task, error) {
	tasks := []types.Task{}
	tp := ecs.NewListTasksPaginator(app.ecs, &ecs.ListTasksInput{Cluster: &cluster})
	for tp.HasMorePages() {
		to, err := tp.NextPage(app.ctx)
		if err != nil {
			return nil, err
		}
		if len(to.TaskArns) == 0 {
			continue
		}
		out, err := app.ecs.DescribeTasks(app.ctx, &ecs.DescribeTasksInput{
			Cluster: &cluster,
			Tasks:   to.TaskArns,
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

func (app *Ecsta) listClusters() ([]string, error) {
	clusters := []string{}
	tp := ecs.NewListClustersPaginator(app.ecs, &ecs.ListClustersInput{})
	for tp.HasMorePages() {
		out, err := tp.NextPage(app.ctx)
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
