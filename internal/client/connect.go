package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type Options struct {
	Protocol, Host, User, Password, Command string
	Port                                    int
}

func Connect(o Options) error {
	if o.Protocol != "ssh" && o.Protocol != "sftp" {
		return fmt.Errorf("protocol must be ssh or sftp")
	}
	if o.Protocol == "sftp" && o.Command != "" {
		return fmt.Errorf("-c is only supported with ssh protocol")
	}
	if o.Host == "" || o.User == "" || o.Password == "" {
		return fmt.Errorf("host, user, and pwd are required")
	}
	if o.Port < 1 || o.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	dir, err := os.MkdirTemp("", "ssh-proxy-askpass-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	askpass := filepath.Join(dir, "askpass")
	if err := os.WriteFile(askpass, []byte("#!/bin/sh\nprintf '%s\\n' \"$SSH_PROXY_PASSWORD\"\n"), 0700); err != nil {
		return err
	}
	bin, args := "ssh", []string{"-tt", "-p", strconv.Itoa(o.Port), o.User + "@" + o.Host}
	if o.Command != "" {
		args = []string{"-p", strconv.Itoa(o.Port), o.User + "@" + o.Host}
	}
	if o.Protocol == "sftp" {
		bin, args = "sftp", []string{"-P", strconv.Itoa(o.Port), o.User + "@" + o.Host}
	}
	if o.Command != "" {
		args = append(args, o.Command)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), "SSH_ASKPASS="+askpass, "SSH_ASKPASS_REQUIRE=force", "SSH_PROXY_PASSWORD="+o.Password, "DISPLAY=ssh-proxy")
	return cmd.Run()
}
