package ecsta

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// TestTCPProxyToLocalhost tests the core TCP proxy functionality
func TestTCPProxyToLocalhost(t *testing.T) {
	// Start a mock backend server on localhost (this simulates Session Manager Plugin)
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Failed to start backend server:", err)
	}
	defer backendListener.Close()

	backendPort := backendListener.Addr().(*net.TCPAddr).Port

	// Start mock backend server that echoes data
	var backendWg sync.WaitGroup
	backendCtx, backendCancel := context.WithCancel(context.Background())
	defer backendCancel()

	backendWg.Add(1)
	go func() {
		defer backendWg.Done()
		for {
			select {
			case <-backendCtx.Done():
				return
			default:
			}

			// Set a short timeout for Accept to allow context checking
			backendListener.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
			conn, err := backendListener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // timeout, check context again
				}
				return // Listener closed or other error
			}

			// Reset deadline for the connection
			backendListener.(*net.TCPListener).SetDeadline(time.Time{})

			go func(c net.Conn) {
				defer c.Close()
				// Echo server: read and write back
				buffer := make([]byte, 1024)
				n, err := c.Read(buffer)
				if err != nil {
					return
				}
				_, err = c.Write(buffer[:n])
				if err != nil {
					return
				}
			}(conn)
		}
	}()

	// Get a different port for proxy frontend
	proxyListener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal("Failed to get proxy port:", err)
	}
	proxyPort := proxyListener.Addr().(*net.TCPAddr).Port
	proxyListener.Close() // Free the port for proxy to use

	// Create Ecsta instance and start TCP proxy
	app := &Ecsta{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var proxyWg sync.WaitGroup
	proxyWg.Add(1)
	go func() {
		defer proxyWg.Done()
		err := app.startTCPProxyToLocalhost(ctx, "0.0.0.0", proxyPort, backendPort)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Logf("Proxy ended with: %v", err)
		}
	}()

	// Wait a bit for proxy to start
	time.Sleep(50 * time.Millisecond)

	// Test client connection to proxy
	clientConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatal("Failed to connect to proxy:", err)
	}
	defer clientConn.Close()

	// Test data transfer
	testData := "Hello, TCP Proxy!"

	// Send data
	_, err = clientConn.Write([]byte(testData))
	if err != nil {
		t.Fatal("Failed to write data:", err)
	}

	// Read echoed data
	buffer := make([]byte, len(testData))
	_, err = io.ReadFull(clientConn, buffer)
	if err != nil {
		t.Fatal("Failed to read data:", err)
	}

	// Verify data
	if string(buffer) != testData {
		t.Errorf("Data mismatch: got %q, want %q", string(buffer), testData)
	}

	// Close client connection before cleanup
	clientConn.Close()

	// Cleanup: cancel contexts and wait for goroutines
	cancel()
	backendCancel()

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		proxyWg.Wait()
		backendWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished successfully
	case <-time.After(2 * time.Second):
		t.Log("Warning: goroutines did not finish within timeout")
	}
}

// TestHandleProxyConnection tests the actual handleProxyConnection function
func TestHandleProxyConnection(t *testing.T) {
	// Start backend server
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Failed to start backend:", err)
	}
	defer backendListener.Close()

	backendPort := backendListener.Addr().(*net.TCPAddr).Port

	// Simple echo server with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			backendListener.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
			conn, err := backendListener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c) // Echo
			}(conn)
		}
	}()

	// Create mock client-server connection pair
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	app := &Ecsta{}
	proxyCtx, proxyCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer proxyCancel()

	// Start handleProxyConnection
	var proxyWg sync.WaitGroup
	proxyWg.Add(1)
	go func() {
		defer proxyWg.Done()
		app.handleProxyConnection(proxyCtx, serverConn, backendPort)
	}()

	// Test communication
	testData := "test data"
	go func() {
		clientConn.Write([]byte(testData))
	}()

	buffer := make([]byte, len(testData))
	_, err = io.ReadFull(clientConn, buffer)
	if err != nil {
		t.Fatal("Failed to read:", err)
	}

	if string(buffer) != testData {
		t.Errorf("Got %q, want %q", string(buffer), testData)
	}

	// Clean shutdown
	clientConn.Close()
	serverConn.Close()
	proxyCancel()
	cancel()

	// Wait for proxy goroutine with timeout
	done := make(chan struct{})
	go func() {
		proxyWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Log("Warning: proxy goroutine did not finish within timeout")
	}
}
