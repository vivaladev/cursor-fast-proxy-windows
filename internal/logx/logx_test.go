package logx

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMaskReplacesLongerSecretFirst(t *testing.T) {
	got := Mask("ssh 203.0.113.10 pass 203.0.113.10-secret", map[string]string{
		"CURSOR_FAST_PROXY_IP":   "203.0.113.10",
		"CURSOR_FAST_PROXY_PASS": "203.0.113.10-secret",
	})
	want := "ssh CURSOR_FAST_PROXY_IP pass CURSOR_FAST_PROXY_PASS"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestMaskSkipsEmpty(t *testing.T) {
	got := Mask("keep", map[string]string{"CURSOR_FAST_PROXY_USER": ""})
	if got != "keep" {
		t.Fatalf("got %q", got)
	}
}

func TestLines(t *testing.T) {
	when := time.Date(2026, 11, 2, 13, 4, 5, 0, time.UTC)
	secrets := map[string]string{"CURSOR_FAST_PROXY_PASS": "p a"}
	if got := CommandLine(when, "run p a", secrets); got != "2026-11-02T13:04:05-run CURSOR_FAST_PROXY_PASS" {
		t.Fatalf("command %q", got)
	}
	if got := ErrorLine(when, "ssh", "fail p a", secrets); got != "2026-11-02T13:04:05-ERROR ssh: fail CURSOR_FAST_PROXY_PASS" {
		t.Fatalf("error %q", got)
	}
}

func TestAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CursorFastProxy", "commands.log")
	if err := Append(path, "one"); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, "two"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "one\ntwo\n" {
		t.Fatalf("got %q", raw)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("log mode %o", info.Mode().Perm())
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatal("неожиданное содержимое")
	}
}
