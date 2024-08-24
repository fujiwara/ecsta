package ecsta

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/schollz/progressbar/v3"
)

type CpOption struct {
	Src      string `arg:"" help:"Source"`
	Dest     string `arg:"" help:"Destination"`
	Port     int    `help:"port number for file transfer" default:"12345"`
	Progress bool   `help:"show progress bar" negatable:"" default:"true"`

	ID        string  `help:"task ID"`
	Container string  `help:"container name"`
	Family    *string `help:"task definition family name"`
	Service   *string `help:"ECS service name"`
}

func (cp *CpOption) SrcTarget() (string, string) {
	return parseCpTarget(cp.Src)
}

func (cp *CpOption) DestTarget() (string, string) {
	return parseCpTarget(cp.Dest)
}

func parseCpTarget(s string) (string, string) {
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		return parts[0], parts[1]
	}
	return "", s
}

//go:embed assets/tncl-x86_64-linux-musl
var agentBinaryX86_64 []byte

//go:embed assets/tncl-aarch64-linux-musl
var agentBinaryARM64 []byte

var bootAgentTmpl = template.Must(template.New("").Parse(`sh -e -c 'cat <<EOF_OF_AGENT_COMMAND | base64 -d > {{.Cmd}}
{{.Base64Binary}}
EOF_OF_AGENT_COMMAND

chmod +x {{.Cmd}}
{{.Cmd}} {{.Port}} {{if .Upload}}>{{else}}<{{end}} {{.Filename}}
'
`))

type bootAgentTmplData struct {
	Base64Binary string
	Cmd          string
	Port         int
	Upload       bool
	Filename     string
}

type cpTask struct {
	taskArn     string
	taskCPUArch string
	container   string
	upload      bool
	localFile   string
	remoteFile  string
	port        int
}

func (cp *cpTask) bootAgent() string {
	buf := &strings.Builder{}
	var b64 string
	switch cp.taskCPUArch {
	case "ARM64":
		b64 = base64.StdEncoding.EncodeToString(agentBinaryARM64)
	case "x86_64":
		b64 = base64.StdEncoding.EncodeToString(agentBinaryX86_64)
	default:
		slog.Warn("unknown CPU architecture", "arch", cp.taskCPUArch)
		b64 = base64.StdEncoding.EncodeToString(agentBinaryX86_64) // default
	}
	bootAgentTmpl.Execute(buf, &bootAgentTmplData{
		Base64Binary: b64,
		Cmd:          "/tmp/tncl",
		Port:         cp.port,
		Upload:       cp.upload,
		Filename:     cp.remoteFile,
	})
	return buf.String()
}

func (app *Ecsta) prepareCp(ctx context.Context, opt *CpOption) (*cpTask, error) {
	cp := &cpTask{
		port: opt.Port,
	}
	srcHost, srcFile := opt.SrcTarget()
	destHost, destFile := opt.DestTarget()
	if strings.HasSuffix(destFile, "/") { // directory
		destFile += filepath.Base(srcFile) // append basename
	}

	switch {
	case srcHost == "" && destHost == "":
		return nil, fmt.Errorf("either source or destination must be remote")
	case srcHost == "": // local -> remote
		slog.Info("cp local to remote", "src", srcFile, "dest", destFile)
		cp.localFile = srcFile
		cp.remoteFile = destFile
		cp.upload = true
		if srcHost != "_" { // task ID
			opt.ID = srcHost
		}
	case destHost == "": // remote -> local
		slog.Info("cp remote to local", "src", srcFile, "dest", destFile)
		cp.localFile = destFile
		cp.remoteFile = srcFile
		cp.upload = false
		if destHost != "_" { // task ID
			opt.ID = destHost
		}
	default:
		return nil, fmt.Errorf("both source and destination must not be remote")
	}

	if err := app.SetCluster(ctx); err != nil {
		return nil, err
	}
	task, err := app.findTask(ctx, &optionFindTask{
		id: opt.ID, family: opt.Family, service: opt.Service,
		selectFunc: selectFuncExcludeStopped,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to select tasks: %w", err)
	}
	for _, attr := range task.Attributes {
		if *attr.Name == "ecs.cpu-architecture" {
			cp.taskCPUArch = *attr.Value
			break
		}
	}

	container, err := app.findContainerName(ctx, task, opt.Container)
	if err != nil {
		return nil, fmt.Errorf("failed to select containers: %w", err)
	}
	cp.taskArn = *task.TaskArn
	cp.container = container
	return cp, nil
}

func (app *Ecsta) RunCp(ctx context.Context, opt *CpOption) error {
	cp, err := app.prepareCp(ctx, opt)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	var succeeded, down atomic.Bool
	succeeded.Store(false)
	down.Store(false)

	agent := &sync.WaitGroup{}
	agent.Add(1)
	// boot agent via exec
	go func(ctx context.Context) {
		defer agent.Done()
		slog.Info("booting agent in the target container", "task", cp.taskArn, "container", cp.container, "port", opt.Port)
		err := app.RunExec(ctx, &ExecOption{
			ID:          cp.taskArn,
			Container:   cp.container,
			Command:     cp.bootAgent(),
			catchSignal: true,
		})
		down.Store(true)
		if err != nil {
			if succeeded.Load() {
				slog.Debug("agent stopped", "error", err)
				return
			}
			slog.Error("failed to boot agent", "error", err)
		}
	}(ctx)
	time.Sleep(3 * time.Second) // wait for agent boot
	// TODO: wait for the agent to be ready
	if down.Load() {
		cancel()
		return fmt.Errorf("agent stopped")
	}

	portforward := &sync.WaitGroup{}
	portforward.Add(1)
	// portforward to the agent
	go func(ctx context.Context) {
		defer portforward.Done()
		slog.Info("starting portforward to the agent", "task", cp.taskArn, "container", cp.container, "port", cp.port)
		err := app.RunPortforward(ctx, &PortforwardOption{
			ID:         cp.taskArn,
			Container:  cp.container,
			LocalPort:  cp.port,
			RemotePort: cp.port,
		})
		if err != nil {
			if succeeded.Load() {
				slog.Debug("portforward stopped", "error", err)
				return
			}
			slog.Error("failed to portforward", "error", err)
		}
	}(ctx)

	// teardown
	defer func() {
		slog.Info("waiting for agent stop...")
		agent.Wait()
		cancel() // stop the portforward after the agent stops
		portforward.Wait()
	}()

	// connect to the agent
	slog.Info("connecting to agent via portforward", "task", cp.taskArn, "container", cp.container, "port", opt.Port)
	client, err := newNcClient("localhost", opt.Port, opt.Progress)
	if err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer client.Close()

	// send/receive file
	if cp.upload {
		slog.Info("start to send", "src", cp.localFile, "dest", cp.remoteFile)
		if err := client.SendFile(cp.localFile); err != nil {
			return err
		}
	} else {
		slog.Info("start to receive", "src", cp.remoteFile, "dest", cp.localFile)
		if err := client.ReceiveFile(cp.localFile); err != nil {
			return err
		}
	}
	succeeded.Store(true)
	return nil
}

type ncClient struct {
	conn     net.Conn
	progress bool
}

func (c *ncClient) Close() error {
	return c.conn.Close()
}

func newNcClient(host string, port int, progress bool) (*ncClient, error) {
	for {
		slog.Info("connecting", "host", host, "port", port)
		conn, err := net.DialTimeout(
			"tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second,
		)
		if err != nil {
			time.Sleep(1 * time.Second)
			slog.Debug("retrying", "error", err)
			continue
		}
		slog.Info("connected", "host", host, "port", port)
		return &ncClient{conn: conn, progress: progress}, nil
	}
}

func (c *ncClient) SendFile(fileName string) error {
	st, err := os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	size := st.Size()
	slog.Info("sending file", "src", fileName, "size", size)
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	var w io.Writer
	if c.progress {
		var bs = size
		if bs == 0 { // progressbar does not support 0 size
			bs = -1 // unknown size
		}
		bar := progressbar.DefaultBytes(bs, "sending")
		w = io.MultiWriter(c.conn, bar)
	} else {
		w = c.conn
	}
	n, err := io.Copy(w, f)
	if err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}
	slog.Info("sent", "src", fileName, "size", n)

	return nil
}

func (c *ncClient) ReceiveFile(fileName string) error {
	slog.Info("receiving file", "dest", fileName)
	var f io.WriteCloser
	if fileName == "-" {
		f = os.Stdout
	} else {
		ff, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		f = ff
	}
	defer f.Close()
	var w io.Writer
	if c.progress {
		bar := progressbar.DefaultBytes(-1, "receiving")
		w = io.MultiWriter(f, bar)
	} else {
		w = f
	}
	n, err := io.Copy(w, c.conn)
	if err != nil {
		return fmt.Errorf("failed to receive: %w", err)
	}
	slog.Info("received", "dest", fileName, "size", n)
	return nil
}
