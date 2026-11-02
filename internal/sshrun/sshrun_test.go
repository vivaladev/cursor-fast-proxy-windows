package sshrun

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestRunSendsScriptAndHidesPassword(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	const password = "s3cret-value"
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if meta.User() == "root" && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("denied")
		},
	}
	cfg.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				serverConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
				if err != nil {
					conn.Close()
					return
				}
				defer serverConn.Close()
				go ssh.DiscardRequests(reqs)
				for newChannel := range chans {
					if newChannel.ChannelType() != "session" {
						_ = newChannel.Reject(ssh.UnknownChannelType, "session only")
						continue
					}
					channel, requests, err := newChannel.Accept()
					if err != nil {
						return
					}
					serveSession(channel, requests)
				}
			}()
		}
	}()

	stdout, err := Run(context.Background(), listener.Addr().String(), "root", password, "marker\n", ssh.FixedHostKey(signer.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "UUID=11111111-1111-4111-8111-111111111111") {
		t.Fatalf("stdout %q", stdout)
	}
	if strings.Contains(stdout, password) {
		t.Fatal("пароль попал в stdout")
	}

	_, err = Run(context.Background(), listener.Addr().String(), "root", password, "nope\n", ssh.FixedHostKey(signer.PublicKey()))
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if strings.Contains(err.Error(), password) || !strings.Contains(err.Error(), "CURSOR_FAST_PROXY_PASS") {
		t.Fatalf("пароль в ошибке: %v", err)
	}
}

func serveSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	for req := range requests {
		if req.Type != "exec" {
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			continue
		}
		var msg struct{ Command string }
		_ = ssh.Unmarshal(req.Payload, &msg)
		_ = req.Reply(true, nil)
		body, _ := io.ReadAll(channel)
		if msg.Command == "bash -s" && strings.Contains(string(body), "marker") && !strings.Contains(string(body), "s3cret") {
			_, _ = io.WriteString(channel, "UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n")
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
			return
		}
		_, _ = io.WriteString(channel.Stderr(), "rejected s3cret-value")
		_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
		return
	}
}

func TestHide(t *testing.T) {
	got := hide("rejected s3cret-value", "s3cret-value")
	if got != "rejected CURSOR_FAST_PROXY_PASS" {
		t.Fatalf("%q", got)
	}
}
