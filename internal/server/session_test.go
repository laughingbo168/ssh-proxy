package server

import (
	"testing"

	"github.com/zhb/ssh-proxy/internal/identity"
)

func TestShellEnvironmentUsesTargetUserHome(t *testing.T) {
	env := shellEnvironment([]string{"HOME=/root", "USER=root", "PATH=/usr/bin"}, &identity.Credential{Name: "user", Home: "/home/user"})
	want := map[string]string{"HOME": "/home/user", "USER": "user", "LOGNAME": "user"}
	for key, value := range want {
		found := false
		for _, entry := range env {
			if entry == key+"="+value {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s=%s in %v", key, value, env)
		}
	}
}
