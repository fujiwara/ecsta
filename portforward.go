package ecsta

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/google/subcommands"
)

type PortforwardCmd struct {
	app *Ecsta

	id         string
	container  string
	localPort  int
	remotePort int
	remoteHost string
}

func NewPortforwardCmd(app *Ecsta) *PortforwardCmd {
	return &PortforwardCmd{
		app: app,
	}
}

func (*PortforwardCmd) Name() string     { return "portforward" }
func (*PortforwardCmd) Synopsis() string { return "port forwarding" }
func (*PortforwardCmd) Usage() string {
	return `portforward [options]:
  Port forwarding to a task in a cluster.
`
}

func (p *PortforwardCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.id, "id", "", "task ID")
	f.StringVar(&p.container, "container", "", "container name")
	f.IntVar(&p.localPort, "localport", 0, "local port")
	f.IntVar(&p.remotePort, "port", 0, "remote port")
	f.StringVar(&p.remoteHost, "host", "", "remote host")
}

func (p *PortforwardCmd) execute(ctx context.Context) error {
	if p.remotePort == 0 {
		return fmt.Errorf("--port is required")
	}
	if p.localPort == 0 {
		return fmt.Errorf("--localport is required")
	}

	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := p.app.findTask(ctx, p.id)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	name, err := p.app.findContainerName(ctx, task, p.container)
	if err != nil {
		return fmt.Errorf("failed to select containers: %w", err)
	}
	p.container = name

	ssmReq, err := buildSsmRequestParameters(task, p.container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}
	endpoint, err := p.app.Endpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	in := &ssm.StartSessionInput{
		Target:       aws.String(ssmReq.Target),
		DocumentName: aws.String("AWS-StartPortForwardingSession"),
		Parameters: map[string][]string{
			"portNumber":      {strconv.Itoa(p.remotePort)},
			"localPortNumber": {strconv.Itoa(p.localPort)},
		},
		Reason: aws.String("port forwarding"),
	}
	if p.remoteHost != "" {
		in.Parameters["host"] = []string{p.remoteHost}
		in.DocumentName = aws.String("AWS-StartPortForwardingSessionToRemoteHost")
	}
	res, err := p.app.ssm.StartSession(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	sess, _ := json.Marshal(res)

	cmd := exec.Command(
		SessionManagerPluginBinary,
		string(sess),
		p.app.region,
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

func (p *PortforwardCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
