//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/apppaths"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/scripts"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/version"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/workflow"
)

func main() {
	if err := runWindow(newEngine()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newEngine() *workflow.Engine {
	home, err := os.UserHomeDir()
	if err != nil {
		home = `C:\Users\Public`
	}
	roaming := os.Getenv("APPDATA")
	if roaming == "" {
		roaming = filepath.Join(home, "AppData", "Roaming")
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(home, "AppData", "Local")
	}
	return &workflow.Engine{
		Paths: apppaths.Paths{
			Home:    home,
			Roaming: roaming,
			Local:   local,
		},
		Bootstrap: scripts.Bootstrap,
		Teardown:  scripts.Teardown,
	}
}

func saveSettings(eng *workflow.Engine, ip, user, password string) error {
	text, err := settings.Write(settings.Credentials{IP: ip, User: user, Password: password})
	if err != nil {
		return err
	}
	path := eng.Paths.Settings()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o600)
}

func loadSettings(eng *workflow.Engine) (settings.Credentials, error) {
	raw, err := os.ReadFile(eng.Paths.Settings())
	if err != nil {
		if os.IsNotExist(err) {
			return settings.Credentials{}, nil
		}
		return settings.Credentials{}, err
	}
	creds, ok, err := settings.Read(string(raw))
	if err != nil || !ok {
		return settings.Credentials{}, err
	}
	return creds, nil
}

func readLog(eng *workflow.Engine) string {
	raw, err := os.ReadFile(eng.Paths.CommandLog())
	if err != nil {
		if os.IsNotExist(err) {
			return "Журнал пока пуст."
		}
		return err.Error()
	}
	if len(raw) > 64*1024 {
		raw = raw[len(raw)-64*1024:]
	}
	return string(raw)
}

func applyProxy(eng *workflow.Engine) error {
	return eng.Apply(context.Background())
}

func removeProxy(eng *workflow.Engine, alsoVPS bool) error {
	return eng.Remove(context.Background(), alsoVPS)
}

func windowTitle() string {
	return fmt.Sprintf("Cursor fast proxy %s", version.Current)
}
