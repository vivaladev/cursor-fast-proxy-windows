package apppaths

import (
	"path/filepath"
	"strings"
)

type Paths struct {
	Home    string
	Roaming string
	Local   string
}

func (p Paths) Settings() string {
	return filepath.Join(p.Home, ".cursor-fast-proxy", "settings.env")
}

func (p Paths) KnownHosts() string {
	return filepath.Join(p.Home, ".cursor-fast-proxy", "known_hosts")
}

func (p Paths) EgressDir() string {
	return filepath.Join(p.Home, "cursor-egress")
}

func (p Paths) ClientConfig() string {
	return filepath.Join(p.EgressDir(), "sing-box-client.json")
}

func (p Paths) SingBox() string {
	return filepath.Join(p.Local, "CursorFastProxy", "bin", "sing-box.exe")
}

func (p Paths) RunnerCmd() string {
	return filepath.Join(p.Local, "CursorFastProxy", "run-singbox.cmd")
}

func (p Paths) TaskXML() string {
	return filepath.Join(p.Local, "CursorFastProxy", "task.xml")
}

func (p Paths) CursorSettings() string {
	return filepath.Join(p.Roaming, "Cursor", "User", "settings.json")
}

func (p Paths) CursorArgv() string {
	return filepath.Join(p.Roaming, "Cursor", "argv.json")
}

func (p Paths) CommandLog() string {
	return filepath.Join(p.Local, "CursorFastProxy", "commands.log")
}

func (p Paths) SingBoxLog() string {
	return filepath.Join(p.Local, "CursorFastProxy", "sing-box.log")
}

func (p Paths) SingBoxErr() string {
	return filepath.Join(p.Local, "CursorFastProxy", "sing-box.err")
}

func (p Paths) Display(path string) string {
	home := filepath.Clean(p.Home)
	clean := filepath.Clean(path)
	if clean == home {
		return "~"
	}
	rel, err := filepath.Rel(home, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return "~" + string(filepath.Separator) + rel
}
