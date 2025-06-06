package ecsta

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/fujiwara/sloghandler"
	"github.com/mattn/go-isatty"
)

type CLI struct {
	Cluster         string `help:"ECS cluster name" short:"c" env:"ECS_CLUSTER"`
	Region          string `help:"AWS region" short:"r" env:"AWS_REGION"`
	Output          string `help:"output format (table, tsv, json)" short:"o" default:"table" enum:"table,tsv,json" env:"ECSTA_OUTPUT"`
	TaskFormatQuery string `help:"A jq query to format task in selector" short:"q" env:"ECSTA_TASK_FORMAT_QUERY"`
	Debug           bool   `help:"enable debug output" env:"ECSTA_DEBUG"`
	LogFormat       string `help:"log format (text, json)" short:"l" default:"text" enum:"text,json" env:"ECSTA_LOG_FORMAT"`

	Configure   *ConfigureOption   `cmd:"" help:"Create a configuration file of ecsta"`
	Describe    *DescribeOption    `cmd:"" help:"Describe tasks"`
	Exec        *ExecOption        `cmd:"" help:"Execute a command on a task"`
	List        *ListOption        `cmd:"" help:"List tasks"`
	Logs        *LogsOption        `cmd:"" help:"Show log messages of a task"`
	Portforward *PortforwardOption `cmd:"" help:"Forward a port of a task"`
	Stop        *StopOption        `cmd:"" help:"Stop a task"`
	Trace       *TraceOption       `cmd:"" help:"Trace a task"`
	CP          *CpOption          `cmd:"" help:"Copy files from/to a task"`
	Version     struct{}           `cmd:"" help:"Show version"`
}

func RunCLI(ctx context.Context, args []string) error {
	var cli CLI
	parser, err := kong.New(&cli, kong.Vars{"version": Version})
	if err != nil {
		return err
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	// Initialize logger with sloghandler
	var handler slog.Handler
	level := slog.LevelInfo
	if cli.Debug {
		level = slog.LevelDebug
	}

	switch cli.LogFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	default: // text
		// Enable color only if stderr is a terminal
		useColor := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
		handler = sloghandler.NewLogHandler(
			os.Stderr,
			&sloghandler.HandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					Level: level,
				},
				Color: useColor,
			},
		)
	}
	slog.SetDefault(slog.New(handler))

	app, err := New(ctx, cli.Region, cli.Cluster)
	if err != nil {
		return err
	}
	app.Config.OverrideCLI(&cli)
	cmd := strings.Fields(kctx.Command())[0]
	return app.Dispatch(ctx, cmd, &cli)
}

func (app *Ecsta) Dispatch(ctx context.Context, command string, cli *CLI) error {
	switch command {
	case "configure":
		return app.RunConfigure(ctx, cli.Configure)
	case "describe":
		return app.RunDescribe(ctx, cli.Describe)
	case "exec":
		return app.RunExec(ctx, cli.Exec)
	case "list":
		return app.RunList(ctx, cli.List)
	case "logs":
		return app.RunLogs(ctx, cli.Logs)
	case "portforward":
		return app.RunPortforward(ctx, cli.Portforward)
	case "stop":
		return app.RunStop(ctx, cli.Stop)
	case "trace":
		return app.RunTrace(ctx, cli.Trace)
	case "cp":
		return app.RunCp(ctx, cli.CP)
	case "version":
		fmt.Printf("ecsta %s\n", Version)
		return nil
	}
	return fmt.Errorf("unknown command: %s", command)
}
