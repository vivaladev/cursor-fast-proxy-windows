package scripts

import (
	"regexp"
	"strings"
	"testing"
)

func TestBootstrapLeavesSSHAndAmneziaAlone(t *testing.T) {
	if regexp.MustCompile(`systemctl[[:space:]]+(restart|reload)[[:space:]]+ssh`).MatchString(Bootstrap) {
		t.Fatal("скрипт трогает ssh")
	}
	if strings.Contains(Bootstrap, "/etc/amnezia") || strings.Contains(Bootstrap, "amnezia") {
		t.Fatal("скрипт трогает amnezia")
	}
	for _, needle := range []string{
		`SINGBOX_VERSION="1.14.0"`,
		`"listen_port": 8443`,
		"xtls-rprx-vision",
		"www.cloudflare.com",
		`KEYS="$CONF_DIR/keys.env"`,
		"enable --now sing-box-cursor",
		"ufw allow 8443/tcp",
		"printf 'UUID=%s",
	} {
		if !strings.Contains(Bootstrap, needle) {
			t.Fatalf("нет %s", needle)
		}
	}
	if strings.Contains(Bootstrap, `"listen_port": 443`) {
		t.Fatal("sidecar слушает 443")
	}
	if strings.Contains(Bootstrap, "tun") {
		t.Fatal("в скрипте есть tun")
	}
}

func TestTeardownLeavesBinarySSHAndAmnezia(t *testing.T) {
	if regexp.MustCompile(`systemctl[[:space:]]+(restart|reload)[[:space:]]+ssh|amnezia|/usr/local/bin/sing-box`).MatchString(Teardown) {
		t.Fatal("teardown трогает ssh, amnezia или бинарь sing-box")
	}
	for _, needle := range []string{
		"disable --now sing-box-cursor",
		"daemon-reload",
		`rm -rf "$CONF_DIR"`,
		"ufw delete allow 8443/tcp",
	} {
		if !strings.Contains(Teardown, needle) {
			t.Fatalf("нет %s", needle)
		}
	}
}

func TestEmbeddedScriptsAreLF(t *testing.T) {
	for name, script := range map[string]string{"bootstrap": Bootstrap, "teardown": Teardown} {
		if strings.Contains(script, "\r") {
			t.Fatalf("%s содержит CR", name)
		}
		if !strings.Contains(script, "set -euo pipefail\n") {
			t.Fatalf("%s: после pipefail нет LF", name)
		}
	}
}

func TestUnixStripsCRLF(t *testing.T) {
	got := Unix("set -euo pipefail\r\ntrue || true\r\nfi\r")
	if strings.Contains(got, "\r") || got != "set -euo pipefail\ntrue || true\nfi\n" {
		t.Fatalf("%q", got)
	}
}

func TestScriptsAreNonEmpty(t *testing.T) {
	if len(Bootstrap) < 500 || len(Teardown) < 100 {
		t.Fatal("скрипты пустые")
	}
}
