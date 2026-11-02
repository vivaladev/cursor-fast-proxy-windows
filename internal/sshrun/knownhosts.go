package sshrun

import (
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func Remember(path string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		host := hostname
		if parsed, _, err := net.SplitHostPort(hostname); err == nil {
			host = parsed
		}
		want := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			name, rest, ok := strings.Cut(line, " ")
			if !ok || name != host {
				continue
			}
			if rest == want {
				return nil
			}
			return fmt.Errorf("ключ %s изменился", host)
		}
		if err := os.MkdirAll(parentDir(path), 0o700); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(host + " " + want + "\n")
		return err
	}
}

func parentDir(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i <= 0 {
		return "."
	}
	return path[:i]
}
