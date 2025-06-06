package ecsta

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type PortforwardOption struct {
	ID          string  `help:"task ID"`
	Container   string  `help:"container name"`
	LocalPort   int     `help:"local port"`
	RemotePort  int     `help:"remote port"`
	RemoteHost  string  `help:"remote host"`
	L           string  `name:"L" help:"short expression of local-port:remote-host:remote-port" short:"L"`
	Family      *string `help:"task definition family name"`
	Service     *string `help:"ECS service name"`
	Public      bool    `help:"bind to all interfaces (0.0.0.0) instead of localhost only"`

	stdout io.Writer
	stderr io.Writer
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

	// Determine local port for Session Manager Plugin
	var ssmLocalPort int
	if opt.Public {
		// Get a specific ephemeral port for Session Manager Plugin when binding to all interfaces
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("failed to find available port: %w", err)
		}
		ssmLocalPort = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	} else {
		// Use user-specified port for normal case
		ssmLocalPort = opt.LocalPort
	}

	in := &ssm.StartSessionInput{
		Target:       aws.String(target),
		DocumentName: aws.String("AWS-StartPortForwardingSession"),
		Parameters: map[string][]string{
			"portNumber":      {strconv.Itoa(opt.RemotePort)},
			"localPortNumber": {strconv.Itoa(ssmLocalPort)},
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

	// Start TCP proxy if public access is requested
	if opt.Public {
		slog.Warn("TCP proxy will bind to all interfaces (0.0.0.0) - ensure proper network security", "port", opt.LocalPort)
		slog.Info("Session Manager Plugin will use port", "port", ssmLocalPort)
		
		// Start TCP proxy in background
		go func() {
			if err := app.startTCPProxyToLocalhost(ctx, "0.0.0.0", opt.LocalPort, ssmLocalPort); err != nil {
				slog.Error("TCP proxy failed", "error", err)
			}
		}()
		
		// Wait a bit for proxy to start
		time.Sleep(200 * time.Millisecond)
	}

	// Run Session Manager Plugin (common path)
	return app.runSessionManagerPlugin(ctx, &task, sess, target, opt.stdout, opt.stderr)
}

// startTCPProxyToLocalhost starts a TCP proxy that listens on bindAddress:frontendPort and forwards to 127.0.0.1:backendPort
func (app *Ecsta) startTCPProxyToLocalhost(ctx context.Context, bindAddress string, frontendPort, backendPort int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindAddress, frontendPort))
	if err != nil {
		return fmt.Errorf("failed to listen on %s:%d: %w", bindAddress, frontendPort, err)
	}
	defer listener.Close()

	slog.Info("TCP proxy listening", "address", fmt.Sprintf("%s:%d", bindAddress, frontendPort), "backend", fmt.Sprintf("127.0.0.1:%d", backendPort))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			slog.Error("failed to accept connection", "error", err)
			continue
		}

		go app.handleProxyConnection(ctx, conn, backendPort)
	}
}

// handleProxyConnection handles a single proxy connection
func (app *Ecsta) handleProxyConnection(ctx context.Context, clientConn net.Conn, port int) {
	defer clientConn.Close()

	// Connect to Session Manager Plugin on localhost
	backendConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		slog.Error("failed to connect to backend", "backend", fmt.Sprintf("127.0.0.1:%d", port), "error", err)
		return
	}
	defer backendConn.Close()

	slog.Debug("proxying connection", "client", clientConn.RemoteAddr(), "backend", fmt.Sprintf("127.0.0.1:%d", port))

	// Start bidirectional copy with context cancellation
	var wg sync.WaitGroup
	done := make(chan struct{})
	
	wg.Add(2)

	// Copy from client to backend
	go func() {
		defer wg.Done()
		if _, err := io.Copy(backendConn, clientConn); err != nil {
			slog.Debug("client to backend copy ended", "error", err)
		}
		backendConn.Close() // Close backend to stop the other direction
	}()

	// Copy from backend to client
	go func() {
		defer wg.Done()
		if _, err := io.Copy(clientConn, backendConn); err != nil {
			slog.Debug("backend to client copy ended", "error", err)
		}
		clientConn.Close() // Close client to stop the other direction
	}()

	// Wait for copy completion or context cancellation
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Debug("proxy connection closed normally", "client", clientConn.RemoteAddr())
	case <-ctx.Done():
		slog.Debug("proxy connection closed by context", "client", clientConn.RemoteAddr(), "error", ctx.Err())
		// Close connections to stop io.Copy operations
		clientConn.Close()
		backendConn.Close()
		wg.Wait()
	}
}

