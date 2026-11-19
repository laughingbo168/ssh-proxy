package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaultsShellFiltersCIDRsAndResolvesRelativeHostKey(t *testing.T) {
	path := writeConfig(t, "listen: 127.0.0.1:2222\nhost_key_path: key\nallow_cidrs: [127.0.0.1/32]\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Shell != "/bin/sh" {
		t.Fatalf("shell = %q", cfg.Shell)
	}
	if cfg.HostKeyPath != filepath.Join(filepath.Dir(path), "key") {
		t.Fatalf("host key = %q", cfg.HostKeyPath)
	}
	if !cfg.Allows(net.ParseIP("127.0.0.1")) || cfg.Allows(net.ParseIP("192.0.2.1")) {
		t.Fatal("unexpected allow result")
	}
}

func TestLoadRejectsInvalidFields(t *testing.T) {
	if _, err := Load(writeConfig(t, "listen: bad\nhost_key_path: ''\nallow_cidrs: [bad]\n")); err == nil {
		t.Fatal("wanted validation error")
	}
}
