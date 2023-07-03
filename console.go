package ecsta

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
)

type ConsoleOption struct{}

type Console struct {
	SelectCluster *SelectClusterOption `cmd:"" name:"cluster" optional:"" help:"Select a cluster"`
	Describe      *DescribeOption      `cmd:"" help:"Describe tasks"`
	Exec          *ExecOption          `cmd:"" help:"Execute a command on a task"`
	List          *ListOption          `cmd:"" help:"List tasks" aliases:"ls"`
	Logs          *LogsOption          `cmd:"" help:"Show log messages of a task"`
	Portforward   *PortforwardOption   `cmd:"" help:"Forward a port of a task"`
	SelectTask    *SelectTaskOption    `cmd:"" name:"task" help:"Select a task"`
	Stop          *StopOption          `cmd:"" help:"Stop a task"`
	Trace         *TraceOption         `cmd:"" help:"Trace a task"`

	Exit struct{} `cmd:"" help:"Exit console" aliases:"quit"`
	Help struct{} `cmd:"" help:"Show help"`
}

type SelectClusterOption struct {
	ClusterName string `arg:"" help:"Cluster name" optional:""`
}

type ConsoleState struct {
	Cluster       string
	TaskID        string
	ClustersCache []string
	TasksCache    []string
}

func (s *ConsoleState) Prompt() string {
	var prompt string
	if s.Cluster != "" {
		prompt = s.Cluster
	}
	if s.TaskID != "" {
		prompt = fmt.Sprintf("%s@%s", s.TaskID, s.Cluster)
	}
	return prompt + "> "
}

func (s *ConsoleState) Reset() {
	s.Cluster = ""
	s.TaskID = ""
	s.ClustersCache = nil
	s.TasksCache = nil
}

func (app *Ecsta) newConsoleCompleter(ctx context.Context, s *ConsoleState) readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("cluster", readline.PcItemDynamic(func(line string) []string {
			if s.ClustersCache != nil {
				return s.ClustersCache
			}
			clusters, err := app.listClusters(ctx)
			if err != nil {
				log.Println("[error]", err)
			}
			names := make([]string, 0, len(clusters))
			for _, cluster := range clusters {
				names = append(names, arnToName(cluster))
			}
			s.ClustersCache = names
			return names
		})),
		readline.PcItem("describe"),
		readline.PcItem("exec"),
		readline.PcItem("help"),
		readline.PcItem("--help"),
		readline.PcItem("list"),
		readline.PcItem("logs"),
		readline.PcItem("portforward"),
		readline.PcItem("stop"),
		readline.PcItem("trace"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("task", readline.PcItemDynamic(func(line string) []string {
			if s.TasksCache != nil {
				return s.TasksCache
			}
			tasks, err := app.listTasks(ctx, &optionListTasks{})
			if err != nil {
				log.Println("[error]", err)
			}
			var names []string
			for _, task := range tasks {
				names = append(names, arnToName(*task.TaskArn))
			}
			s.TasksCache = names
			return names
		})),
	)
}

func (app *Ecsta) RunConsole(ctx context.Context, opt *ConsoleOption) error {
	origCtx := ctx
	ctx, cancel := context.WithCancel(origCtx)
	defer cancel()

	if err := prepareConsoleHistory(); err != nil {
		log.Println("[warn]", err)
	}

	s := &ConsoleState{
		Cluster: app.cluster,
	}
	showHelp := false
	rd, err := readline.NewEx(&readline.Config{
		Prompt:            s.Prompt(),
		HistoryFile:       filepath.Join(xdg.StateDir, "history"),
		AutoComplete:      app.newConsoleCompleter(ctx, s),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer rd.Close()
	readline.CaptureExitSignal(func() {
		cancel()
		ctx, cancel = context.WithCancel(origCtx)
	})

	var console Console
	parser, err := kong.New(&console, kong.Vars{"version": Version})
	parser.Exit = func(int) { showHelp = true }
	if err != nil {
		return err
	}

INPUT:
	for {
		rd.SetPrompt(s.Prompt())
		showHelp = false

		line, err := rd.Readline()
		if err != nil {
			switch err {
			case readline.ErrInterrupt:
				if len(line) == 0 {
					break INPUT
				} else {
					continue INPUT
				}
			case io.EOF:
				break INPUT
			default:
				log.Println("[error]", err)
			}
			continue INPUT
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		args, err := shellwords.Parse(line)
		if err != nil {
			log.Println("[error]", err)
			continue
		}
		if len(args) == 1 && args[0] == "help" {
			// workaround for kong
			args[0] = "--help"
		}
		kctx, err := parser.Parse(args)
		if err != nil {
			log.Println("[error]", err)
			continue
		}
		cmd := strings.Fields(kctx.Command())[0]
		if showHelp {
			continue
		}
		if err := app.DispatchConsole(ctx, cmd, &console, s); err != nil {
			if err == io.EOF {
				break
			}
			log.Println("[error]", err)
		}
	}
	return nil
}

func (app *Ecsta) DispatchConsole(ctx context.Context, command string, console *Console, s *ConsoleState) error {
	switch command {
	case "exit", "quit":
		return io.EOF
	case "help":
		return fmt.Errorf("use --help")
	case "cluster":
		return app.RunSelectCluster(ctx, console.SelectCluster, s)
	}
	if s.Cluster == "" {
		return fmt.Errorf("no cluster is selected. use `cluster` command")
	}

	switch command {
	case "list":
		return app.RunList(ctx, console.List)
	case "task":
		return app.RunSelectTask(ctx, console.SelectTask, s)
	}
	if s.TaskID == "" {
		return fmt.Errorf("no task is selected. use `task` command")
	}

	switch command {
	case "describe":
		console.Describe.ID = s.TaskID
		return app.RunDescribe(ctx, console.Describe)
	case "exec":
		console.Exec.ID = s.TaskID
		return app.RunExec(ctx, console.Exec)
	case "logs":
		console.Logs.ID = s.TaskID
		return app.RunLogs(ctx, console.Logs)
	case "portforward":
		console.Portforward.ID = s.TaskID
		return app.RunPortforward(ctx, console.Portforward)
	case "stop":
		console.Stop.ID = s.TaskID
		return app.RunStop(ctx, console.Stop)
	case "trace":
		console.Trace.ID = s.TaskID
		return app.RunTrace(ctx, console.Trace)
	}
	return fmt.Errorf("unknown command: %s", command)
}

type SelectTaskOption struct {
	TaskID  string  `arg:"" optional:"" help:"task ID or prefix"`
	Family  *string `help:"task definition family name"`
	Service *string `help:"ECS service name"`
}

func (app *Ecsta) RunSelectCluster(ctx context.Context, opt *SelectClusterOption, s *ConsoleState) error {
	if opt.ClusterName == "" {
		app.cluster = ""
		if err := app.SetCluster(ctx); err != nil {
			return nil
		}
		s.Reset()
		s.Cluster = app.cluster
		return nil
	} else {
		c, err := app.getCluster(ctx, opt.ClusterName)
		if err != nil {
			return err
		}
		s.Reset()
		s.Cluster = aws.ToString(c.ClusterName)
		app.cluster = s.Cluster
		return nil
	}
}

func (app *Ecsta) RunSelectTask(ctx context.Context, opt *SelectTaskOption, s *ConsoleState) error {
	switch {
	case opt.TaskID == "":
		task, err := app.findTask(ctx, &optionFindTask{})
		if err != nil {
			return err
		}
		s.TaskID = arnToName(*task.TaskArn)
		return nil
	case len(opt.TaskID) >= 32 && len(opt.TaskID) <= 36:
		_ts, err := app.describeTasks(ctx, &optionDescribeTasks{ids: []string{opt.TaskID}})
		if err != nil {
			return err
		} else if len(_ts) == 1 {
			s.TaskID = arnToName(*_ts[0].TaskArn)
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
			s.TaskID = foundTaskIDs[0]
			return nil
		} else {
			return fmt.Errorf("[error] taskID %s is ambiguous", opt.TaskID)
		}
	}
}
