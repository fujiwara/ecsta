package ecsta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	task, err := app.findTask(ctx, &optionFindTask{id: opt.ID, family: opt.Family, service: opt.Service})
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	name, err := app.findContainerName(ctx, task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to select containers: %w", err)
	}
	opt.Container = name

	ssmReq, err := buildSsmRequestParameters(task, opt.Container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}
	endpoint, err := app.Endpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	in := &ssm.StartSessionInput{
		Target:       aws.String(ssmReq.Target),
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
	sess, _ := json.Marshal(res)

	cmd := exec.Command(
		SessionManagerPluginBinary,
		string(sess),
		app.region,
		"StartSession",
		"",
		ssmReq.String(),
		endpoint,
	)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
