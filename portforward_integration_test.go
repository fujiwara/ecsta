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
	backendWg.Add(1)
	go func() {
		defer backendWg.Done()
		for {
			conn, err := backendListener.Accept()
			if err != nil {
				return // Listener closed
			}
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var proxyWg sync.WaitGroup
	proxyWg.Add(1)
	go func() {
		defer proxyWg.Done()
		err := app.startTCPProxyToLocalhost(ctx, "0.0.0.0", proxyPort, backendPort)
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Proxy failed: %v", err)
		}
	}()

	// Wait a bit for proxy to start
	time.Sleep(100 * time.Millisecond)

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

	// Cleanup
	cancel()
	proxyWg.Wait()
	backendWg.Wait()
}