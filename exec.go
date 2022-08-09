package ecsta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/urfave/cli/v2"
)

const SessionManagerPluginBinary = "session-manager-plugin"

type ExecOption struct {
	ID        string
	Command   string
	Container string
}

func newExecCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "exec",
		Usage: "exec task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "task ID",
			},
			&cli.StringFlag{
				Name:  "command",
				Usage: "command to execute",
				Value: "sh",
			},
			&cli.StringFlag{
				Name:  "container",
				Usage: "container name",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunExec(c.Context, &ExecOption{
				ID:        c.String("id"),
				Command:   c.String("command"),
				Container: c.String("container"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunExec(ctx context.Context, opt *ExecOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := app.findTask(ctx, opt.ID)
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
	sess, _ := json.Marshal(out.Session)
	ssmReq, err := buildSsmRequestParameters(task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}
	endpoint, err := app.Endpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	cmd := exec.Command(
		SessionManagerPluginBinary,
		string(sess),
		app.region,
		"StartSession",
		"",
		ssmReq.String(),
		endpoint,
	)
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
