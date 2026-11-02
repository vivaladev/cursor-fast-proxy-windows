package apppaths

import (
	"path/filepath"
	"testing"
)

func TestLayout(t *testing.T) {
	p := Paths{Home: filepath.Join("C:", "Users", "me"), Roaming: filepath.Join("C:", "Users", "me", "AppData", "Roaming"), Local: filepath.Join("C:", "Users", "me", "AppData", "Local")}
	if filepath.Base(p.ClientConfig()) != "sing-box-client.json" {
		t.Fatal(p.ClientConfig())
	}
	if filepath.Base(p.Settings()) != "settings.env" {
		t.Fatal(p.Settings())
	}
	if filepath.Base(filepath.Dir(p.CursorSettings())) != "User" {
		t.Fatal(p.CursorSettings())
	}
	if filepath.Base(p.CursorArgv()) != "argv.json" {
		t.Fatal(p.CursorArgv())
	}
	shown := p.Display(p.Settings())
	if shown[0] != '~' {
		t.Fatalf("display %s", shown)
	}
}
