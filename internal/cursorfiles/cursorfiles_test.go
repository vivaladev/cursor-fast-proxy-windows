package cursorfiles

import (
	"strings"
	"testing"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

func TestApplyAndRemoveSettingsKeepsOtherKeys(t *testing.T) {
	source := "{\n  // theme\n  \"workbench.colorTheme\": \"dark\"\n}\n"
	applied, err := ApplySettings(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range canon.SettingsKeys {
		if !strings.Contains(applied, `"`+key+`"`) {
			t.Fatalf("нет ключа %s в %s", key, applied)
		}
	}
	if !strings.Contains(applied, canon.ProxyURL) || !strings.Contains(applied, "override") || !strings.Contains(applied, "true") {
		t.Fatalf("значения не записаны: %s", applied)
	}
	if !strings.Contains(applied, "workbench.colorTheme") || !strings.Contains(applied, "// theme") {
		t.Fatalf("чужие данные пропали: %s", applied)
	}
	removed, err := RemoveSettings(applied)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range canon.SettingsKeys {
		if strings.Contains(removed, `"`+key+`"`) {
			t.Fatalf("ключ %s остался: %s", key, removed)
		}
	}
	if !strings.Contains(removed, "workbench.colorTheme") || !strings.Contains(removed, "// theme") {
		t.Fatalf("после удаления пропало чужое: %s", removed)
	}
}

func TestApplyArgvAndRemove(t *testing.T) {
	applied, err := ApplyArgv("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(applied, canon.ProxyURL) || strings.Contains(applied, "proxy-bypass-list") {
		t.Fatalf("argv %s", applied)
	}
	removed, err := RemoveArgv(applied + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(removed, "proxy-server") || strings.Contains(removed, "proxy-bypass-list") {
		t.Fatalf("argv keys остались: %s", removed)
	}
}

func TestApplyDropsExistingBypass(t *testing.T) {
	settings, err := ApplySettings("{\n  \"http.noProxy\": \"localhost,.example\"\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(settings, "noProxy") || strings.Contains(settings, ".example") {
		t.Fatalf("обход остался: %s", settings)
	}
	argv, err := ApplyArgv("{\n  \"proxy-bypass-list\": \"<local>\"\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(argv, "bypass") || strings.Contains(argv, "<local>") {
		t.Fatalf("обход argv остался: %s", argv)
	}
}
