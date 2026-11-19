package server

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/zhb/ssh-proxy/internal/identity"
	"golang.org/x/crypto/ssh"
)

func handleSession(ch ssh.Channel, reqs <-chan *ssh.Request, shell string, user *identity.Credential) {
	var cols, rows uint32
	var gotPTY bool
	var started bool
	var mu sync.Mutex
	var ptmx *os.File
	_ = ptmx
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			if len(req.Payload) < 8 {
				_ = req.Reply(false, nil)
				continue
			}
			termLen := int(binary.BigEndian.Uint32(req.Payload[:4]))
			if len(req.Payload) < 4+termLen+8 {
				_ = req.Reply(false, nil)
				continue
			}
			cols = binary.BigEndian.Uint32(req.Payload[4+termLen:])
			rows = binary.BigEndian.Uint32(req.Payload[8+termLen:])
			gotPTY = true
			_ = req.Reply(true, nil)
		case "window-change":
			if len(req.Payload) >= 8 {
				cols = binary.BigEndian.Uint32(req.Payload[:4])
				rows = binary.BigEndian.Uint32(req.Payload[4:])
				mu.Lock()
				if ptmx != nil {
					_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
				}
				mu.Unlock()
			}
			_ = req.Reply(true, nil)
		case "shell":
			if !gotPTY || started {
				_ = req.Reply(false, nil)
				continue
			}
			started = true
			cmd := exec.Command(shell, "-l")
			if user != nil {
				if uint32(os.Geteuid()) != user.UID || uint32(os.Getegid()) != user.GID {
					cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: user.UID, Gid: user.GID}}
				}
				cmd.Dir = user.Home
				cmd.Env = shellEnvironment(os.Environ(), user)
			}
			p, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
			if err != nil {
				log.Printf("failed to start shell %q: %v", shell, err)
				_ = req.Reply(false, nil)
				return
			}
			mu.Lock()
			ptmx = p
			mu.Unlock()
			_ = req.Reply(true, nil)
			go io.Copy(p, ch)
			_, _ = io.Copy(ch, p)
			_ = p.Close()
			_ = cmd.Wait()
			_, _ = ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
			_ = ch.Close()
			return
		case "subsystem":
			var request struct{ Name string }
			if err := ssh.Unmarshal(req.Payload, &request); err != nil || request.Name != "sftp" || started {
				_ = req.Reply(false, nil)
				continue
			}
			started = true
			_ = req.Reply(true, nil)
			serveSFTP(ch)
			return
		case "exec":
			var request struct{ Command string }
			if err := ssh.Unmarshal(req.Payload, &request); err != nil || started {
				_ = req.Reply(false, nil)
				continue
			}
			started = true
			cmd := exec.Command(shell, "-lc", request.Command)
			if user != nil {
				cmd.Dir, cmd.Env = user.Home, shellEnvironment(os.Environ(), user)
			}
			cmd.Stdout, cmd.Stderr = ch, ch
			_ = req.Reply(true, nil)
			err := cmd.Run()
			status := 0
			if exit, ok := err.(*exec.ExitError); ok {
				status = exit.ExitCode()
			} else if err != nil {
				status = 1
			}
			_, _ = ch.SendRequest("exit-status", false, []byte{0, 0, 0, byte(status)})
			_ = ch.Close()
			return
		default:
			_ = req.Reply(false, nil)
		}
	}
}

func shellEnvironment(base []string, user *identity.Credential) []string {
	if user == nil {
		return base
	}
	result := make([]string, 0, len(base)+3)
	for _, entry := range base {
		if strings.HasPrefix(entry, "HOME=") || strings.HasPrefix(entry, "USER=") || strings.HasPrefix(entry, "LOGNAME=") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "HOME="+user.Home, "USER="+user.Name, "LOGNAME="+user.Name)
}
