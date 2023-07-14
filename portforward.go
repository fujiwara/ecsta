package ecsta

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type PortforwardOption struct {
	ID         string  `help:"task ID"`
	Container  string  `help:"container name"`
	LocalPort  int     `help:"local port" required:"true"`
	RemotePort int     `help:"remote port" required:"true"`
	RemoteHost string  `help:"remote host"`
	Family     *string `help:"task definiton family name"`
	Service    *string `help:"ECS service name"`
}

func (app *Ecsta) RunPortforward(ctx context.Context, opt *PortforwardOption) error {
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

	target, err := ssmRequestTarget(task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}

	in := &ssm.StartSessionInput{
		Target:       aws.String(target),
		DocumentName: aws.String("AWS-StartPortForwardingSession"),
		Parameters: map[string][]string{
			"portNumber":      {strconv.Itoa(opt.RemotePort)},
			"localPortNumber": {strconv.Itoa(opt.LocalPort)},
		},
		Reason: aws.String("port forwarding"),
	}
	if opt.RemoteHost != "" {
		in.Parameters["host"] = []string{opt.RemoteHost}
		in.DocumentName = aws.String("AWS-StartPortForwardingSessionToRemoteHost")
	}
	res, err := app.ssm.StartSession(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	sess := &types.Session{
		SessionId:  res.SessionId,
		StreamUrl:  res.StreamUrl,
		TokenValue: res.TokenValue,
	}
	return app.runSessionManagerPlugin(ctx, task, sess, target)
}
