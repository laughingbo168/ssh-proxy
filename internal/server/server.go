package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net"
	"sync"

	"github.com/zhb/ssh-proxy/internal/config"
	"github.com/zhb/ssh-proxy/internal/identity"
	"golang.org/x/crypto/ssh"
)

func validateUsername(actual, expected string) error {
	if expected != "" && actual != expected {
		return errors.New("Username does not exist or access is denied")
	}
	return nil
}

func validatePassword(actual []byte, expected string) error {
	if subtle.ConstantTimeCompare(actual, []byte(expected)) != 1 {
		return errors.New("Username does not exist or access is denied")
	}
	return nil
}

type Server struct {
	cfg      config.Config
	signer   ssh.Signer
	log      *slog.Logger
	user     *identity.Credential
	password string
	mu       sync.Mutex
	listener net.Listener
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
}

func New(cfg config.Config, signer ssh.Signer, log *slog.Logger, user *identity.Credential, password string) *Server {
	return &Server{cfg: cfg, signer: signer, log: log, user: user, password: password, conns: map[net.Conn]struct{}{}}
}
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}
func (s *Server) Serve(l net.Listener) error {
	s.mu.Lock()
	s.listener = l
	s.mu.Unlock()
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		s.wg.Add(1)
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		go s.handle(c)
	}
}
func (s *Server) handle(c net.Conn) {
	defer s.wg.Done()
	defer c.Close()
	defer func() { s.mu.Lock(); delete(s.conns, c); s.mu.Unlock() }()
	host, _, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil || !s.cfg.Allows(net.ParseIP(host)) {
		s.log.Warn("rejected connection", "remote", c.RemoteAddr())
		return
	}
	sc := &ssh.ServerConfig{NoClientAuth: s.password == ""}
	if s.password != "" {
		sc.PasswordCallback = func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if err := validateUsername(meta.User(), expectedUsername(s.user)); err != nil {
				return nil, err
			}
			if err := validatePassword(password, s.password); err != nil {
				return nil, err
			}
			return nil, nil
		}
	}
	sc.AddHostKey(s.signer)
	conn, chans, reqs, err := ssh.NewServerConn(c, sc)
	if err != nil {
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	if s.password == "" && s.user != nil {
		if err := validateUsername(conn.User(), expectedUsername(s.user)); err != nil {
			s.log.Warn("rejected SSH username", "username", conn.User(), "reason", err)
			for ch := range chans {
				_ = ch.Reject(ssh.Prohibited, err.Error())
				return
			}
			return
		}
	}
	for ch := range chans {
		if ch.ChannelType() != "session" {
			_ = ch.Reject(ssh.UnknownChannelType, "interactive sessions only")
			continue
		}
		channel, requests, err := ch.Accept()
		if err == nil {
			go handleSession(channel, requests, s.cfg.Shell, s.user)
		}
	}
}

func expectedUsername(user *identity.Credential) string {
	if user == nil {
		return ""
	}
	return user.Name
}
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.listener != nil {
		_ = s.listener.Close()
	}
	for c := range s.conns {
		_ = c.Close()
	}
	s.mu.Unlock()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
