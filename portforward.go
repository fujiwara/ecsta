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
	"github.com/urfave/cli/v2"
)

type PortforwardOption struct {
	ID         string
	Container  string
	LocalPort  int
	RemotePort int
	RemoteHost string
}

func newPortforwardCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "portforward",
		Usage: "forward port to task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "task ID",
			},
			&cli.StringFlag{
				Name:  "container",
				Usage: "container name",
			},
			&cli.IntFlag{
				Name:  "local-port",
				Usage: "local port",
				Value: 0,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "remote port",
				Value: 0,
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "remote host",
			},
		},
		Action: func(c *cli.Context) error {
			app, err := NewFromCLI(c)
			if err != nil {
				return err
			}
			return app.RunPortforward(c.Context, &PortforwardOption{
				ID:         c.String("id"),
				Container:  c.String("container"),
				LocalPort:  c.Int("local-port"),
				RemotePort: c.Int("port"),
				RemoteHost: c.String("host"),
			})
		},
	}
	cmd.Flags = append(cmd.Flags, globalFlags...)
	return cmd
}

func (app *Ecsta) RunPortforward(ctx context.Context, opt *PortforwardOption) error {
	if opt.RemotePort == 0 {
		return fmt.Errorf("--port is required")
	}
	if opt.LocalPort == 0 {
		return fmt.Errorf("--localport is required")
	}

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
