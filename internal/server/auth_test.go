package server

import "testing"

func TestValidateUsername(t *testing.T) {
	if err := validateUsername("user", "user"); err != nil {
		t.Fatal(err)
	}
	if err := validateUsername("adm", "user"); err == nil {
		t.Fatal("expected username rejection")
	}
}

func TestUsernameErrorExplainsExpectedUser(t *testing.T) {
	err := validateUsername("adm", "user")
	if err == nil || err.Error() != "Username does not exist or access is denied" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword([]byte("secret"), "secret"); err != nil {
		t.Fatal(err)
	}
	if err := validatePassword([]byte("bad"), "secret"); err == nil {
		t.Fatal("expected password rejection")
	}
}
