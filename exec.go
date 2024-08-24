package ecsta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/creack/pty"
	"github.com/mattn/go-isatty"
)

const SessionManagerPluginBinary = "session-manager-plugin"

type ExecOption struct {
	ID        string  `help:"task ID"`
	Command   string  `help:"command to execute" default:"sh"`
	Container string  `help:"container name"`
	Family    *string `help:"task definition family name"`
	Service   *string `help:"ECS service name"`

	catchSignal bool
}

func (app *Ecsta) RunExec(ctx context.Context, opt *ExecOption) error {
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

	name, err := app.findContainerName(ctx, task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to select containers: %w", err)
	}
	opt.Container = name

	out, err := app.ecs.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     task.ClusterArn,
		Interactive: true,
		Task:        task.TaskArn,
		Command:     optional(opt.Command),
		Container:   optional(opt.Container),
	})
	if err != nil {
		return fmt.Errorf("failed to execute command. %w See also https://github.com/aws-containers/amazon-ecs-exec-checker", err)
	}
	target, err := ssmRequestTarget(task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}

	if !opt.catchSignal {
		signal.Ignore(os.Interrupt)
	}
	return app.runSessionManagerPlugin(ctx, task, out.Session, target)
}

func (app *Ecsta) runSessionManagerPlugin(ctx context.Context, task types.Task, session *types.Session, target string) error {
	endpoint, err := app.Endpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}
	sess, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	ssmreq, err := json.Marshal(map[string]string{
		"Target": target,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal ssm request parameters: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		SessionManagerPluginBinary,
		string(sess),
		app.region,
		"StartSession",
		"",
		string(ssmreq),
		endpoint,
	)
	// send SIGINT to session manager plugin the context is canceled.
	cmd.Cancel = func() error {
		slog.Info(fmt.Sprintf("sending SIGINT to %s", SessionManagerPluginBinary))
		return cmd.Process.Signal(os.Interrupt)
	}
	// send SIGKILL after 3 seconds if SIGINT is ignored.
	cmd.WaitDelay = 3 * time.Second

	go func() {
		if err := app.watchTaskUntilStopping(ctx, *task.TaskArn); err != nil {
			slog.Info(err.Error())
			cancel()
		}
	}()
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else {
		slog.Info("running in non-interactive mode (tty is not available)")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			return fmt.Errorf("failed to start pty: %w", err)
		}
		defer ptmx.Close()
		go func() {
			io.Copy(os.Stdout, ptmx)
		}()
		return cmd.Wait()
	}
}

func (app *Ecsta) watchTaskUntilStopping(ctx context.Context, taskID string) error {
	ticker := time.NewTicker(10 * time.Second) // TODO: configurable
	defer ticker.Stop()
	var lastStatus string
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		tasks, err := app.describeTasks(ctx, &optionDescribeTasks{
			ids: []string{taskID},
		})
		if err != nil {
			continue
		}
		if len(tasks) == 0 {
			return fmt.Errorf("task not found: %s", taskID)
		}
		status := aws.ToString(tasks[0].LastStatus)
		switch status {
		case "STOPPING", "DEPROVISIONING", "STOPPED", "DELETED":
			return fmt.Errorf(
				"%s is %s: %s (%s)",
				taskID,
				status,
				tasks[0].StopCode,
				aws.ToString(tasks[0].StoppedReason),
			)
		case "DEACTIVATING":
			if lastStatus != status {
				slog.Warn(
					"the task will be stopped",
					"task_id", taskID,
					"status", status,
					"stop_code", tasks[0].StopCode,
				)
			}
		}
		lastStatus = status
	}
}
