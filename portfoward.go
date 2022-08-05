package ecsta

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/google/subcommands"
)

type PortfowardCmd struct {
	app *Ecsta

	id        string
	container string
	command   string
}

func NewPortfowardCmd(app *Ecsta) *PortfowardCmd {
	return &PortfowardCmd{
		app: app,
	}
}

func (*PortfowardCmd) Name() string     { return "exec" }
func (*PortfowardCmd) Synopsis() string { return "exec task" }
func (*PortfowardCmd) Usage() string {
	return `exec [options]:
  ECS Exec task in a cluster.
`
}

func (p *PortfowardCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.id, "id", "", "task ID")
	f.StringVar(&p.command, "command", "sh", "command")
	f.StringVar(&p.container, "container", "", "container name")
}

func (p *PortfowardCmd) execute(ctx context.Context) error {
	if err := p.app.SetCluster(ctx); err != nil {
		return err
	}
	task, err := p.app.findTask(ctx, p.id)
	if err != nil {
		return fmt.Errorf("failed to select tasks: %w", err)
	}

	if p.container == "" {
		if len(task.Containers) == 1 {
			p.container = *task.Containers[0].Name
		} else {
			containerNames := make([]string, 0, len(task.Containers))
			for _, container := range task.Containers {
				containerNames = append(containerNames, *container.Name)
			}
			container, err := p.app.selectByFilter(ctx, containerNames)
			if err != nil {
				return err
			}
			p.container = container
		}
	}

	out, err := p.app.ecs.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     task.ClusterArn,
		Interactive: true,
		Task:        task.TaskArn,
		Command:     &p.command,
		Container:   &p.container,
	})
	if err != nil {
		return fmt.Errorf("failed to execute command. %w See also https://github.com/aws-containers/amazon-ecs-exec-checker", err)
	}
	sess, _ := json.Marshal(out.Session)
	ssmReq, err := buildSsmRequestParameters(task, p.container)
	if err != nil {
		return fmt.Errorf("failed to build ssm request parameters: %w", err)
	}
	endpoint, err := p.app.Endpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	cmd := exec.Command(
		SessionManagerPluginBinary,
		string(sess),
		p.app.region,
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

func (p *PortfowardCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := p.execute(ctx); err != nil {
		log.Println("[error]", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitFailure
}
