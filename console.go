package ecsta

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
)

type ConsoleOption struct{}

type Console struct {
	Describe    *DescribeOption    `cmd:"" help:"Describe tasks"`
	Exec        *ExecOption        `cmd:"" help:"Execute a command on a task"`
	List        *ListOption        `cmd:"" help:"List tasks" aliases:"ls"`
	Logs        *LogsOption        `cmd:"" help:"Show log messages of a task"`
	Portforward *PortforwardOption `cmd:"" help:"Forward a port of a task"`
	Select      *SelectOption      `cmd:"" help:"Select a task"`
	Stop        *StopOption        `cmd:"" help:"Stop a task"`
	Trace       *TraceOption       `cmd:"" help:"Trace a task"`
	Exit        struct{}           `cmd:"" help:"Exit console" aliases:"quit"`
	Help        struct{}           `cmd:"" help:"Show help"`
}

var consoleCompleter = readline.NewPrefixCompleter(
	readline.PcItem("describe"),
	readline.PcItem("exec"),
	readline.PcItem("help"),
	readline.PcItem("--help"),
	readline.PcItem("list"),
	readline.PcItem("logs"),
	readline.PcItem("portforward"),
	readline.PcItem("select"),
	readline.PcItem("stop"),
	readline.PcItem("trace"),
	readline.PcItem("exit"),
	readline.PcItem("quit"),
)

func (app *Ecsta) RunConsole(ctx context.Context, opt *ConsoleOption) error {
	if err := app.SetCluster(ctx); err != nil {
		return err
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            fmt.Sprintf("%s> ", app.cluster),
		HistoryFile:       filepath.Join(os.Getenv("HOME"), ".local/state/ecsta/history"),
		AutoComplete:      consoleCompleter,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer l.Close()
	l.CaptureExitSignal()

	var console Console
	var showHelp bool
	parser, err := kong.New(&console, kong.Vars{"version": Version})
	parser.Exit = func(int) { showHelp = true }
	if err != nil {
		return err
	}

	var taskID string
INPUT:
	for {
		if taskID == "" {
			l.SetPrompt(fmt.Sprintf("%s> ", app.cluster))
		} else {
			l.SetPrompt(fmt.Sprintf("%s@%s> ", taskID, app.cluster))
		}
		showHelp = false

		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		args, err := shellwords.Parse(line)
		if err != nil {
			log.Println("[error]", err)
			continue INPUT
		}
		kctx, err := parser.Parse(args)
		if err != nil {
			log.Println("[error]", err)
			continue INPUT
		}
		cmd := strings.Fields(kctx.Command())[0]
		if showHelp {
			log.Println("[debug]", cmd, showHelp)
			continue INPUT
		}
		if err := app.DispatchConsole(ctx, cmd, &console, &taskID); err != nil {
			if err == io.EOF {
				break
			}
			log.Println("[error]", err)
		}
	}
	return nil
}

func (app *Ecsta) DispatchConsole(ctx context.Context, command string, console *Console, taskID *string) error {
	switch command {
	case "exit", "quit":
		return io.EOF
	case "help":
		return fmt.Errorf("use --help")
	case "list":
		return app.RunList(ctx, console.List)
	case "select":
		return app.RunSelect(ctx, console.Select, taskID)
	}

	if *taskID == "" {
		return fmt.Errorf("task ID is not specified. use `select` command")
	}

	switch command {
	case "describe":
		console.Describe.ID = *taskID
		return app.RunDescribe(ctx, console.Describe)
	case "exec":
		console.Exec.ID = *taskID
		return app.RunExec(ctx, console.Exec)
	case "logs":
		console.Logs.ID = *taskID
		return app.RunLogs(ctx, console.Logs)
	case "portforward":
		console.Portforward.ID = *taskID
		return app.RunPortforward(ctx, console.Portforward)
	case "stop":
		console.Stop.ID = *taskID
		return app.RunStop(ctx, console.Stop)
	case "trace":
		console.Trace.ID = *taskID
		return app.RunTrace(ctx, console.Trace)
	}
	return fmt.Errorf("unknown command: %s", command)
}

type SelectOption struct {
	TaskID  string  `arg:"" optional:"" help:"task ID or prefix"`
	Family  *string `help:"task definition family name"`
	Service *string `help:"ECS service name"`
}

func (app *Ecsta) RunSelect(ctx context.Context, opt *SelectOption, selected *string) error {
	switch {
	case opt.TaskID == "":
		task, err := app.findTask(ctx, &optionFindTask{})
		if err != nil {
			return err
		}
		*selected = arnToName(*task.TaskArn)
		return nil
	case len(opt.TaskID) >= 32 && len(opt.TaskID) <= 36:
		_ts, err := app.describeTasks(ctx, &optionDescribeTasks{ids: []string{opt.TaskID}})
		if err != nil {
			return err
		} else if len(_ts) == 1 {
			*selected = arnToName(*_ts[0].TaskArn)
			return nil
		} else {
			return fmt.Errorf("taskID %s not found", opt.TaskID)
		}
	default:
		tasks, err := app.listTasks(ctx, &optionListTasks{})
		if err != nil {
			return err
		}
		foundTaskIDs := []string{}
		for _, t := range tasks {
			id := arnToName(*t.TaskArn)
			if strings.HasPrefix(id, opt.TaskID) {
				foundTaskIDs = append(foundTaskIDs, id)
			}
		}
		if len(foundTaskIDs) == 0 {
			return fmt.Errorf("taskID %s not found", opt.TaskID)
		} else if len(foundTaskIDs) == 1 {
			*selected = foundTaskIDs[0]
			return nil
		} else {
			return fmt.Errorf("[error] taskID %s is ambiguous", opt.TaskID)
		}
	}
}
