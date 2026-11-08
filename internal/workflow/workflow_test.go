package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/apppaths"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
)

func testEngine(t *testing.T) (*Engine, string) {
	t.Helper()
	root := t.TempDir()
	moment := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	eng := &Engine{
		Paths: apppaths.Paths{
			Home:    root,
			Roaming: filepath.Join(root, "Roaming"),
			Local:   filepath.Join(root, "Local"),
		},
		Now:       func() time.Time { return moment },
		Sleep:     func(ctx context.Context, d time.Duration) error { moment = moment.Add(d); return nil },
		Bootstrap: "bootstrap-marker\n",
		Teardown:  "teardown-marker\n",
		Schtasks:  "schtasks",
		Netstat:   "netstat",
	}
	return eng, root
}

func writeSettings(t *testing.T, eng *Engine) {
	t.Helper()
	text, err := settings.Write(settings.Credentials{IP: "203.0.113.10", User: "alice", Password: "pw secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(eng.Paths.Settings()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eng.Paths.Settings(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSSHSendsLFToBash(t *testing.T) {
	eng, _ := testEngine(t)
	var got string
	eng.SSH = func(ctx context.Context, creds settings.Credentials, script string) (string, error) {
		got = script
		return "", nil
	}
	_, err := eng.ssh(context.Background(), settings.Credentials{}, "set -euo pipefail\r\ntrue || true\r\nfi\r")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "\r") || got != "set -euo pipefail\ntrue || true\nfi\n" {
		t.Fatalf("bash получил %q", got)
	}
}

func TestApplyStopsBeforeDownloadWhenSettingsMissing(t *testing.T) {
	eng, _ := testEngine(t)
	called := false
	eng.Download = func(ctx context.Context, dest string) error { called = true; return nil }
	err := eng.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "настройки") {
		t.Fatal(err)
	}
	if called {
		t.Fatal("скачивание началось без настроек")
	}
}

func TestApplyHappyPathMasksSecrets(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	var script string
	eng.SSH = func(ctx context.Context, creds settings.Credentials, got string) (string, error) {
		script = got
		if strings.Contains(got, "pw secret") {
			t.Fatal("пароль ушёл в скрипт")
		}
		return "noise\nUUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n", nil
	}
	eng.Download = func(ctx context.Context, dest string) error {
		return os.WriteFile(dest, []byte("exe"), 0o755)
	}
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) {
		if name == "netstat" {
			return "  TCP    127.0.0.1:18081        0.0.0.0:0              LISTENING       4\n", nil
		}
		return "", nil
	}
	if err := eng.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if script != eng.Bootstrap {
		t.Fatalf("script %q", script)
	}
	raw, err := os.ReadFile(eng.Paths.ClientConfig())
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "tun") || strings.Contains(text, "0.0.0.0") || !strings.Contains(text, "127.0.0.1") {
		t.Fatalf("client %s", text)
	}
	info, err := os.Stat(eng.Paths.ClientConfig())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	settingsText, err := os.ReadFile(eng.Paths.CursorSettings())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settingsText), "http://127.0.0.1:18081") || !strings.Contains(string(settingsText), "disableHttp2") {
		t.Fatalf("settings %s", settingsText)
	}
	logText, err := os.ReadFile(eng.Paths.CommandLog())
	if err != nil {
		t.Fatal(err)
	}
	log := string(logText)
	if strings.Contains(log, "pw secret") || strings.Contains(log, "PUBLICKEYPUBLICKEY") || strings.Contains(log, "alice@") {
		t.Fatalf("лог раскрыл данные: %s", log)
	}
	if !strings.Contains(log, "CURSOR_FAST_PROXY_USER") || !strings.Contains(log, "CURSOR_FAST_PROXY_IP") {
		t.Fatalf("лог не замаскирован: %s", log)
	}
	if _, err := os.Stat(eng.Paths.SingBox()); err != nil {
		t.Fatal(err)
	}
}

func TestApplyDoesNotDownloadWhenBinaryExists(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	if err := os.MkdirAll(filepath.Dir(eng.Paths.SingBox()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eng.Paths.SingBox(), []byte("already"), 0o755); err != nil {
		t.Fatal(err)
	}
	eng.Download = func(ctx context.Context, dest string) error { t.Fatal("download"); return nil }
	eng.SSH = func(ctx context.Context, creds settings.Credentials, script string) (string, error) {
		return "UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n", nil
	}
	eng.Run = readyNetstat
	if err := eng.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRetriesTaskOnce(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	eng.Download = func(ctx context.Context, dest string) error { return os.WriteFile(dest, []byte("exe"), 0o755) }
	eng.SSH = func(ctx context.Context, creds settings.Credentials, script string) (string, error) {
		return "UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n", nil
	}
	creates := 0
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "/Create") {
			creates++
			if creates == 1 {
				return "", fmt.Errorf("ERROR: The task is already running")
			}
		}
		if name == "netstat" {
			return "  TCP    127.0.0.1:18081        0.0.0.0:0              LISTENING       4\n", nil
		}
		return "", nil
	}
	if err := eng.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if creates != 2 {
		t.Fatalf("create %d", creates)
	}
}

func TestExposedListenerFailsAfterCursorWrite(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	eng.Download = func(ctx context.Context, dest string) error { return os.WriteFile(dest, []byte("exe"), 0o755) }
	eng.SSH = func(ctx context.Context, creds settings.Credentials, script string) (string, error) {
		return "UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n", nil
	}
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) {
		if name == "netstat" {
			return "  TCP    0.0.0.0:18081         0.0.0.0:0              LISTENING       9\n", nil
		}
		return "", nil
	}
	err := eng.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "проверка") {
		t.Fatal(err)
	}
	raw, readErr := os.ReadFile(eng.Paths.CursorSettings())
	if readErr != nil || !strings.Contains(string(raw), "http.proxy") {
		t.Fatalf("cursor %s %v", raw, readErr)
	}
}

func TestListenTimeout(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	eng.Download = func(ctx context.Context, dest string) error { return os.WriteFile(dest, []byte("exe"), 0o755) }
	eng.SSH = func(ctx context.Context, creds settings.Credentials, script string) (string, error) {
		return "UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef12\n", nil
	}
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) { return "", nil }
	err := eng.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "не слушает") {
		t.Fatal(err)
	}
}

func TestRemoveKeepsSettingsBinaryAndStripsCursor(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	if err := os.MkdirAll(filepath.Dir(eng.Paths.SingBox()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eng.Paths.SingBox(), []byte("exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(eng.Paths.EgressDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eng.Paths.ClientConfig(), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(eng.Paths.EgressDir(), "notes.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	cursor := "{\n  // theme\n  \"workbench.colorTheme\": \"dark\",\n  \"http.proxy\": \"http://127.0.0.1:18081\"\n}\n"
	if err := os.MkdirAll(filepath.Dir(eng.Paths.CursorSettings()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eng.Paths.CursorSettings(), []byte(cursor), 0o600); err != nil {
		t.Fatal(err)
	}
	argv := "{\n  \"locale\": \"ru\",\n  \"proxy-server\": \"http://127.0.0.1:18081\"\n}\n"
	if err := os.WriteFile(eng.Paths.CursorArgv(), []byte(argv), 0o600); err != nil {
		t.Fatal(err)
	}
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) { return "", nil }
	if err := eng.Remove(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(eng.Paths.ClientConfig()); !os.IsNotExist(err) {
		t.Fatal("клиентский конфиг остался")
	}
	if _, err := os.Stat(filepath.Join(eng.Paths.EgressDir(), "notes.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(eng.Paths.SingBox()); err != nil {
		t.Fatal("бинарь удалён")
	}
	if _, err := os.Stat(eng.Paths.Settings()); err != nil {
		t.Fatal("настройки удалены")
	}
	left, err := os.ReadFile(eng.Paths.CursorSettings())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(left), "http.proxy") || !strings.Contains(string(left), "workbench.colorTheme") || !strings.Contains(string(left), "// theme") {
		t.Fatalf("settings %s", left)
	}
	argvLeft, err := os.ReadFile(eng.Paths.CursorArgv())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(argvLeft), "proxy-server") || !strings.Contains(string(argvLeft), "locale") {
		t.Fatalf("argv %s", argvLeft)
	}
}

func TestRemoveVPSUsesTeardownScript(t *testing.T) {
	eng, _ := testEngine(t)
	writeSettings(t, eng)
	var script string
	eng.SSH = func(ctx context.Context, creds settings.Credentials, got string) (string, error) {
		script = got
		return "", nil
	}
	eng.Run = func(ctx context.Context, name string, args ...string) (string, error) { return "", nil }
	if err := eng.Remove(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if script != eng.Teardown {
		t.Fatalf("script %q", script)
	}
}

func readyNetstat(ctx context.Context, name string, args ...string) (string, error) {
	if name == "netstat" {
		return "  TCP    127.0.0.1:18081        0.0.0.0:0              LISTENING       4\n", nil
	}
	return "", nil
}
