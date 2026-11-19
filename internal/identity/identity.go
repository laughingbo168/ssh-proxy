package identity

import (
	"fmt"
	"os/user"
	"strconv"
)

type Credential struct {
	UID, GID uint32
	Home     string
	Name     string
}

func Resolve(name string) (*Credential, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("lookup user %q: %w", name, err)
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	return &Credential{UID: uint32(uid), GID: uint32(gid), Home: u.HomeDir, Name: u.Username}, nil
}
