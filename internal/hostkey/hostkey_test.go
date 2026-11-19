package hostkey

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreatePersistsKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "host_ed25519")
	a, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(a.PublicKey().Marshal()) != string(b.PublicKey().Marshal()) {
		t.Fatal("key changed")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%#o", info.Mode().Perm())
	}
}
