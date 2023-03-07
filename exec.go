package ecsta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

const SessionManagerPluginBinary = "session-manager-plugin"

type ExecOption struct {
	ID        string  `help:"task ID"`
	Command   string  `help:"command to execute" default:"sh"`
	Container string  `help:"container name"`
	Family    *string `help:"task definiton family name"`
	Service   *string `help:"ECS service name"`
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
