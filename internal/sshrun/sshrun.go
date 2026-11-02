package sshrun

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/logx"
	"golang.org/x/crypto/ssh"
)

func Run(ctx context.Context, address, user, password, script string, callback ssh.HostKeyCallback) (string, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: callback,
		Timeout:         0,
	}
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", fmt.Errorf("ssh: %s", hide(err.Error(), password))
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("ssh: %s", hide(err.Error(), password))
	}
	client := ssh.NewClient(c, chans, reqs)
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh: %s", hide(err.Error(), password))
	}
	defer session.Close()
	session.Stdin = strings.NewReader(script)
	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run("bash -s"); err != nil {
		return "", fmt.Errorf("ssh: %s", hide(strings.TrimSpace(stderr.String()+" "+err.Error()), password))
	}
	return stdout.String(), nil
}

func hide(text, password string) string {
	if password == "" {
		return text
	}
	return logx.Mask(text, map[string]string{"CURSOR_FAST_PROXY_PASS": password})
}
