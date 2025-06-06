package ecsta

import (
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