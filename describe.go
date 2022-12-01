package ecsta

import (
	"context"
	"fmt"
)

type DescribeOption struct {
	ID string `help:"task ID"`
}

func (app *Ecsta) RunDescribe(ctx context.Context, opt *DescribeOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, &optionFindTask{id: opt.ID})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}
	f := newTaskFormatterJSON(app.w)
	f.AddTask(task)
	f.Close()
	return nil
}
