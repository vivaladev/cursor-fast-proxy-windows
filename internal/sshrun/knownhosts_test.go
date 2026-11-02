package sshrun

import (
	"crypto/rand"
	"crypto/rsa"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestRememberPinsKey(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer1, err := ssh.NewSignerFromKey(first)
	if err != nil {
		t.Fatal(err)
	}
	signer2, err := ssh.NewSignerFromKey(second)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dir", "known_hosts")
	callback := Remember(path)
	if err := callback("203.0.113.10:22", nil, signer1.PublicKey()); err != nil {
		t.Fatal(err)
	}
	if err := callback("203.0.113.10:22", nil, signer1.PublicKey()); err != nil {
		t.Fatal(err)
	}
	if err := callback("203.0.113.10", nil, signer2.PublicKey()); err == nil {
		t.Fatal("другой ключ принят")
	}
}
