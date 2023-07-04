package ecsta

import (
	"context"
	"fmt"
)

type ListOption struct {
	Family  *string `help:"Task definition family" short:"f"`
	Service *string `help:"Service name" short:"s"`
}

func (app *Ecsta) RunList(ctx context.Context, opt *ListOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	tasks, err := app.listTasks(ctx, &optionListTasks{
		family:  opt.Family,
		service: opt.Service,
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks in cluster %s: %w", app.cluster, err)
	}
	fopt := formatterOption{
		Format:    app.Config.Output,
		HasHeader: true,
	}
	if query := app.Config.TaskFormatQuery; query != "" {
		fopt.Format = "json"
		fopt.Query = query
	}
	f, err := newTaskFormatter(app.w, fopt)
	if err != nil {
		return fmt.Errorf("failed to create task formatter: %w", err)
	}
	for _, task := range tasks {
		f.AddTask(task)
	}
	f.Close()
	return nil
}
