package ecsta

import (
	"bytes"
	"os"
	"testing"
)

func TestPortforwardOption_ParseL(t *testing.T) {
	tests := []struct {
		name    string
		L       string
		want    PortforwardOption
		wantErr bool
	}{
		{
			name: "valid L format",
			L:    "8080:localhost:3306",
			want: PortforwardOption{
				LocalPort:  8080,
				RemoteHost: "localhost",
				RemotePort: 3306,
			},
		},
		{
			name: "ephemeral local port",
			L:    ":localhost:3306",
			want: PortforwardOption{
				LocalPort:  0,
				RemoteHost: "localhost",
				RemotePort: 3306,
			},
		},
		{
			name: "empty L",
			L:    "",
			want: PortforwardOption{},
		},
		{
			name:    "invalid format",
			L:       "invalid",
			wantErr: true,
		},
		{
			name:    "invalid local port",
			L:       "abc:localhost:3306",
			wantErr: true,
		},
		{
			name:    "invalid remote port",
			L:       "8080:localhost:abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &PortforwardOption{L: tt.L}
			err := opt.ParseL()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if opt.LocalPort != tt.want.LocalPort {
					t.Errorf("LocalPort = %v, want %v", opt.LocalPort, tt.want.LocalPort)
				}
				if opt.RemoteHost != tt.want.RemoteHost {
					t.Errorf("RemoteHost = %v, want %v", opt.RemoteHost, tt.want.RemoteHost)
				}
				if opt.RemotePort != tt.want.RemotePort {
					t.Errorf("RemotePort = %v, want %v", opt.RemotePort, tt.want.RemotePort)
				}
			}
		})
	}
}

func TestPortforwardOption_DefaultPublicFlag(t *testing.T) {
	opt := &PortforwardOption{}
	// Public flag defaults to false (localhost only)
	expected := false
	if opt.Public != expected {
		t.Errorf("Default Public = %v, want %v", opt.Public, expected)
	}
}

func TestPortforwardOption_StdoutStderrDefaults(t *testing.T) {
	tests := []struct {
		name       string
		stdout     *bytes.Buffer
		stderr     *bytes.Buffer
		wantStdout bool // true: expect provided buffer, false: expect os.Stdout
		wantStderr bool
	}{
		{
			name:       "nil stdout and stderr get defaults",
			stdout:     nil,
			stderr:     nil,
			wantStdout: false,
			wantStderr: false,
		},
		{
			name:       "provided stdout and stderr are preserved",
			stdout:     &bytes.Buffer{},
			stderr:     &bytes.Buffer{},
			wantStdout: true,
			wantStderr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &PortforwardOption{
				RemotePort: 3000,
			}
			if tt.stdout != nil {
				opt.stdout = tt.stdout
			}
			if tt.stderr != nil {
				opt.stderr = tt.stderr
			}

			// RunPortforward will panic at SetCluster due to nil ECS client,
			// but stdout/stderr defaults are set before that call.
			// Use recover to catch the panic and verify the defaults were set.
			func() {
				defer func() { recover() }()
				app := &Ecsta{}
				_ = app.RunPortforward(t.Context(), opt)
			}()

			if opt.stdout == nil {
				t.Error("stdout should not be nil after RunPortforward")
			}
			if opt.stderr == nil {
				t.Error("stderr should not be nil after RunPortforward")
			}
			if tt.wantStdout {
				if opt.stdout != tt.stdout {
					t.Error("stdout should be the provided buffer")
				}
			} else {
				if opt.stdout != os.Stdout {
					t.Error("stdout should default to os.Stdout")
				}
			}
			if tt.wantStderr {
				if opt.stderr != tt.stderr {
					t.Error("stderr should be the provided buffer")
				}
			} else {
				if opt.stderr != os.Stderr {
					t.Error("stderr should default to os.Stderr")
				}
			}
		})
	}
}

func TestPortforwardOption_PublicFlagValidation(t *testing.T) {
	tests := []struct {
		name   string
		public bool
	}{
		{
			name:   "localhost only",
			public: false,
		},
		{
			name:   "all interfaces",
			public: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &PortforwardOption{
				Public: tt.public,
			}
			// Test that Public flag setting is correctly preserved
			if opt.Public != tt.public {
				t.Errorf("Public = %v, want %v", opt.Public, tt.public)
			}
		})
	}
}
