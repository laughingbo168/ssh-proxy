package server

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func serveSFTP(channel ssh.Channel) {
	server, err := sftp.NewServer(channel)
	if err != nil {
		_ = channel.Close()
		return
	}
	_ = server.Serve()
	_ = server.Close()
	_ = channel.Close()
}
