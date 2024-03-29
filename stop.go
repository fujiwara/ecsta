package ecsta

import (
	"context"
	"fmt"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type StopOption struct {
	ID      string  `help:"task ID"`
	Force   bool    `help:"stop without confirmation"`
	Family  *string `help:"task definition family name"`
	Service *string `help:"ECS service name"`
}

func (app *Ecsta) RunStop(ctx context.Context, opt *StopOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, &optionFindTask{
		id: opt.ID, family: opt.Family, service: opt.Service,
		selectFunc: selectFuncExcludeStopped,
	})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	doStop := true
	if !opt.Force {
		doStop = prompter.YN(fmt.Sprintf("Do you request to stop a task %s?", arnToName(*task.TaskArn)), false)
	}
	if !doStop {
		return ErrAborted
	}
	if _, err := app.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &app.cluster,
		Task:    task.TaskArn,
		Reason:  aws.String("Request stop task by user action."),
	}); err != nil {
		return fmt.Errorf("failed to stop task %s: %w", arnToName(*task.TaskArn), err)
	}
	return nil
}
