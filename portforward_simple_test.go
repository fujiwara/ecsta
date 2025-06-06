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

// TestStartTCPProxyToLocalhost tests TCP proxy functionality with separate ports
func TestStartTCPProxyToLocalhost(t *testing.T) {
	// Start backend server on localhost (simulates Session Manager Plugin)
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Failed to start backend server:", err)
	}
	defer backendListener.Close()
	
	backendPort := backendListener.Addr().(*net.TCPAddr).Port
	
	// Start echo backend server
	var backendWg sync.WaitGroup
	backendWg.Add(1)
	go func() {
		defer backendWg.Done()
		for {
			conn, err := backendListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buffer := make([]byte, 1024)
				n, err := c.Read(buffer)
				if err != nil {
					return
				}
				c.Write(buffer[:n]) // Echo back
			}(conn)
		}
	}()
	
	// Get a free port for proxy frontend
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Failed to get proxy port:", err)
	}
	proxyPort := proxyListener.Addr().(*net.TCPAddr).Port
	proxyListener.Close() // Free the port for proxy to use
	
	// Create custom proxy function for testing
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var proxyWg sync.WaitGroup
	proxyWg.Add(1)
	go func() {
		defer proxyWg.Done()
		// Start proxy on proxyPort, forward to backendPort
		listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", proxyPort))
		if err != nil {
			t.Errorf("Failed to start proxy: %v", err)
			return
		}
		defer listener.Close()
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			
			go func(clientConn net.Conn) {
				defer clientConn.Close()
				
				// Connect to backend
				backendConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", backendPort))
				if err != nil {
					return
				}
				defer backendConn.Close()
				
				// Bidirectional copy
				var copyWg sync.WaitGroup
				copyWg.Add(2)
				
				go func() {
					defer copyWg.Done()
					io.Copy(backendConn, clientConn)
				}()
				
				go func() {
					defer copyWg.Done()
					io.Copy(clientConn, backendConn)
				}()
				
				copyWg.Wait()
			}(conn)
		}
	}()
	
	// Wait for proxy to start
	time.Sleep(100 * time.Millisecond)
	
	// Test client connection
	clientConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatal("Failed to connect to proxy:", err)
	}
	defer clientConn.Close()
	
	// Test data transfer
	testData := "Hello, Proxy!"
	_, err = clientConn.Write([]byte(testData))
	if err != nil {
		t.Fatal("Failed to write:", err)
	}
	
	buffer := make([]byte, len(testData))
	_, err = io.ReadFull(clientConn, buffer)
	if err != nil {
		t.Fatal("Failed to read:", err)
	}
	
	if string(buffer) != testData {
		t.Errorf("Data mismatch: got %q, want %q", string(buffer), testData)
	}
	
	// Cleanup
	cancel()
	proxyWg.Wait()
	backendWg.Wait()
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
	
	// Simple echo server
	go func() {
		for {
			conn, err := backendListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	
	// Create mock client-server connection pair
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	
	app := &Ecsta{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Start handleProxyConnection
	go app.handleProxyConnection(ctx, serverConn, backendPort)
	
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
}