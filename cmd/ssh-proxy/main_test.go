package main

import (
	"strings"
	"testing"
)

func TestHelpTextDocumentsServeCommand(t *testing.T) {
	text := helpText()
	for _, snippet := range []string{"ssh-proxy serve -p 2222 -u user", "ssh-proxy init", "-c CONFIG"} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("help missing %q: %s", snippet, text)
		}
	}
}

func TestDefaultConfigSupportsConfiglessStartup(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Listen != "0.0.0.0:2222" || cfg.HostKeyPath != "./host_ed25519" || cfg.Shell != "/bin/sh" {
		t.Fatalf("unexpected default config: %+v", cfg)
	}
}

func TestWithPortPreservesConfiguredHost(t *testing.T) {
	address, err := withPort("0.0.0.0:2200", 2222)
	if err != nil {
		t.Fatal(err)
	}
	if address != "0.0.0.0:2222" {
		t.Fatalf("address = %q", address)
	}
}

func TestWithPortRejectsInvalidPort(t *testing.T) {
	if _, err := withPort("127.0.0.1:2200", 0); err == nil {
		t.Fatal("expected invalid port error")
	}
}
