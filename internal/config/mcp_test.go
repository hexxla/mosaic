package config

import "testing"

func Test_validateLoopbackListenAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "loopback_ipv4", addr: "127.0.0.1:8787"},
		{name: "loopback_ipv6", addr: "[::1]:8787"},
		{name: "localhost_named", addr: "localhost:8787"},
		{name: "omit_host_bad", addr: ":8787", wantErr: true},
		{name: "all_interfaces_bad", addr: "0.0.0.0:8787", wantErr: true},
		{name: "public_bad", addr: "8.8.8.8:53", wantErr: true},
		{name: "missing_port_bad", addr: "127.0.0.1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateLoopbackListenAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateLoopbackListenAddr(%q): err=%v wantErr=%v", tt.addr, err, tt.wantErr)
			}
		})
	}
}

func TestLoadMCPFromEnv_defaults(t *testing.T) {
	t.Setenv(envMCPAddr, "")
	t.Setenv(envMCPPath, "")
	got, err := LoadMCPFromEnv()
	if err != nil {
		t.Fatalf("LoadMCPFromEnv: %v", err)
	}
	want := MCP{Addr: DefaultMCPAddr, Path: DefaultMCPPath}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestLoadMCPFromEnv_path_invalid(t *testing.T) {
	t.Setenv(envMCPAddr, "127.0.0.1:1")
	t.Setenv(envMCPPath, "bad")
	_, err := LoadMCPFromEnv()
	if err == nil {
		t.Fatal("expected error for path without leading slash")
	}
}
