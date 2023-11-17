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

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := app.RunExec(ctx, &ExecOption{
			ID:        *task.TaskArn,
			Container: container,
			Command:   "ecsta agent",
		})
		if err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
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
	}()

	c := grpcp.NewClient(&grpcp.ClientOption{
		Port: 8022, // local port
	})
	for i := 0; i < 10; i++ {
		if _, err := c.Ping(ctx, "127.0.0.1"); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		log.Println("waiting for agent...")
	}
	src := strings.Replace(opt.Src, "_:", "127.0.0.1:", 1)
	dest := strings.Replace(opt.Dest, "_:", "127.0.0.1:", 1)
	if err := c.Copy(ctx, src, dest); err != nil {
		return err
	}
	//	wg.Wait()
	return nil
}
