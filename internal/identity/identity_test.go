package identity

import (
	"os/user"
	"testing"
)

func TestResolveCurrentUser(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	credential, err := Resolve(current.Username)
	if err != nil {
		t.Fatal(err)
	}
	if credential.UID == 0 && current.Uid != "0" {
		t.Fatalf("unexpected uid: %d", credential.UID)
	}
}
