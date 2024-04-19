package ecsta

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type PortforwardOption struct {
	ID         string  `help:"task ID"`
	Container  string  `help:"container name"`
	LocalPort  int     `help:"local port"`
	RemotePort int     `help:"remote port"`
	RemoteHost string  `help:"remote host"`
	L          string  `name:"L" help:"short expression of local-port:remote-host:remote-port" short:"L"`
	Family     *string `help:"task definition family name"`
	Service    *string `help:"ECS service name"`
}

func (opt *PortforwardOption) ParseL() error {
	if opt.L == "" {
		return nil
	}
	parts := strings.SplitN(opt.L, ":", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid format: %s", opt.L)
	}
	if parts[0] != "" {
		localPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid local port: %s", parts[0])
		}
		opt.LocalPort = localPort
	} else {
		opt.LocalPort = 0 // use ephemeral port
	}
	remotePort, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid remote port: %s", parts[2])
	}
	opt.RemoteHost = parts[1]
	opt.RemotePort = remotePort
	return nil
}

func (app *Ecsta) RunPortforward(ctx context.Context, opt *PortforwardOption) error {
	if err := opt.ParseL(); err != nil {
		return err
	}
	if opt.RemotePort == 0 {
		return fmt.Errorf("remote-port must be specified")
	}

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
