package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/zhb/ssh-proxy/internal/client"
	"github.com/zhb/ssh-proxy/internal/config"
	"github.com/zhb/ssh-proxy/internal/hostkey"
	"github.com/zhb/ssh-proxy/internal/identity"
	"github.com/zhb/ssh-proxy/internal/server"
)

const example = "listen: \"0.0.0.0:2222\"\nhost_key_path: \"./host_ed25519\"\nallow_cidrs:\n  - \"127.0.0.1/32\"\nshell: \"/bin/sh\"\n"

func helpText() string {
	return `ssh-proxy — restricted interactive SSH shell proxy

Usage:
  ssh-proxy init [-c CONFIG]
  ssh-proxy serve [-c CONFIG] [-p PORT] [-u LINUX_USER] [--password PASSWORD]
	  ssh-proxy connect -protocol ssh|sftp -host HOST -p PORT -u USER -pwd PASSWORD

Examples:
  ssh-proxy init -c config.yaml
  ssh-proxy serve -p 2222
  sudo ssh-proxy serve -p 2222 -u user
	  sudo ssh-proxy serve -p 2222 -u user --password 'change-me'

Options:
  -c CONFIG       Optional YAML configuration file
  -p PORT         Listening port; overrides configuration
  -u LINUX_USER   Run interactive shells as this Linux user (requires root to switch users)
	  --password, --pwd PASSWORD  Require this password for SSH login (visible in process listings)
  -h, --help      Show this help
`
}

func main() {
	if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help")) {
		fmt.Print(helpText())
		return
	}
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "ssh-proxy:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ssh-proxy <init|serve> -c CONFIG")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(fs.Output(), helpText()) }
	path := fs.String("c", "", "optional configuration path")
	port := fs.Int("p", 0, "override listening port")
	username := fs.String("u", "", "Linux user for interactive shells")
	host := fs.String("host", "", "server host for connect")
	fs.StringVar(host, "h", "", "server host for connect")
	protocol := fs.String("protocol", "ssh", "connect protocol: ssh or sftp")
	password := fs.String("password", "", "require this SSH password (visible in process listings)")
	fs.StringVar(password, "pwd", "", "alias for --password")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	switch args[0] {
	case "init":
		if *path == "" {
			*path = "config.yaml"
		}
		return initConfig(*path)
	case "serve":
		return serve(*path, *port, *username, *password)
	case "connect":
		return client.Connect(client.Options{Protocol: *protocol, Host: *host, Port: *port, User: *username, Password: *password, Command: *path})
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func defaultConfig() config.Config {
	return config.Config{
		Listen:      "0.0.0.0:2222",
		HostKeyPath: "./host_ed25519",
		Shell:       "/bin/sh",
	}
}
func initConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(example), 0o600); err != nil {
		return err
	}
	_, err := hostkey.LoadOrCreate(filepath.Join(filepath.Dir(path), "host_ed25519"))
	return err
}
func withPort(listen string, port int) (string, error) {
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("port must be between 1 and 65535")
	}
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return "", fmt.Errorf("split listen address: %w", err)
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func serve(path string, port int, username, password string) error {
	cfg := defaultConfig()
	var err error
	if path != "" {
		cfg, err = config.Load(path)
		if err != nil {
			return err
		}
	}
	if port != 0 {
		cfg.Listen, err = withPort(cfg.Listen, port)
		if err != nil {
			return err
		}
	}
	log := slog.Default()
	if len(cfg.AllowCIDRs) == 0 {
		log.Warn("NO SOURCE ALLOWLIST CONFIGURED: every network source can open a shell")
	}
	signer, err := hostkey.LoadOrCreate(cfg.HostKeyPath)
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	var target *identity.Credential
	if username != "" {
		target, err = identity.Resolve(username)
		if err != nil {
			return err
		}
		if os.Geteuid() != 0 && uint32(os.Geteuid()) != target.UID {
			return fmt.Errorf("switching to user %q requires running as root", username)
		}
	}
	s := server.New(cfg, signer, log, target, password)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}()
	log.Info("SSH proxy listening", "address", l.Addr())
	err = s.Serve(l)
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
