package ecsta

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fujiwara/grpcp"
)

type CpOption struct {
	Src  string `arg:"" help:"Source path"`
	Dest string `arg:"" help:"Destination path"`

	ID        string  `help:"task ID"`
	Container string  `help:"container name"`
	Family    *string `help:"task definiton family name"`
	Service   *string `help:"ECS service name"`
}

func (app *Ecsta) RunCp(ctx context.Context, opt *CpOption) error {
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

	container, err := app.findContainerName(ctx, task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to select containers: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		err := app.RunExec(ctx, &ExecOption{
			ID:        *task.TaskArn,
			Container: container,
			Command:   "ecsta agent",
			ignoreSignal: false,
		})
		if err != nil {
			log.Println(err)
		}
	}(ctx)

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		err := app.RunPortforward(ctx, &PortforwardOption{
			ID:         *task.TaskArn,
			Container:  container,
			LocalPort:  8022,
			RemotePort: 8022,
		})
		if err != nil {
			log.Println(err)
		}
	}(ctx)

	defer func() { // teardown
		cancel()
		log.Println("[info] waiting for agent stop...")
		time.Sleep(3 * time.Second)
		wg.Wait()
	}()

	localhost := "127.0.0.1"
	client := grpcp.NewClient(&grpcp.ClientOption{
		Host: localhost,
		Port: 8022,
	})
	ticker := time.NewTicker(1 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if _, err := client.Ping(ctx); err == nil {
			break
		}
		log.Println("[info] waiting for remote agent...")
	}

	src := strings.Replace(opt.Src, "_:", "127.0.0.1:", 1)
	dest := strings.Replace(opt.Dest, "_:", "127.0.0.1:", 1)
	return client.Copy(ctx, src, dest)
}
