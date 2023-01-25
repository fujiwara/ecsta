package ecsta

import (
	"context"
	"fmt"
)

type DescribeOption struct {
	ID      string  `help:"task ID"`
	Family  *string `help:"task definition family name"`
	Service *string `help:"ECS service name"`
}

func (app *Ecsta) RunDescribe(ctx context.Context, opt *DescribeOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, &optionFindTask{id: opt.ID, family: opt.Family, service: opt.Service})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	f, err := newTaskFormatterJSON(app.w, formatterOption{})
	if err != nil {
		return err
	}
	f.AddTask(task)
	f.Close()
	return nil
}
