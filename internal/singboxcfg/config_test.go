package singboxcfg

import (
	"strings"
	"testing"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
)

func TestParseBootstrap(t *testing.T) {
	got, err := ParseBootstrap("noise\nUUID=11111111-1111-1111-1111-111111111111\nPUBLIC_KEY=pub\nSHORT_ID=0123456789abcdef\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID == "" || got.PublicKey != "pub" || got.ShortID != "0123456789abcdef" {
		t.Fatalf("%+v", got)
	}
}

func TestClientStaysOnLoopback(t *testing.T) {
	raw, err := Client(settings.Credentials{IP: "203.0.113.10"}, Bootstrap{
		UUID:      "11111111-1111-1111-1111-111111111111",
		PublicKey: "pub",
		ShortID:   "0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "0.0.0.0") || strings.Contains(text, `"tun"`) {
		t.Fatalf("конфиг \n%s", text)
	}
	if !strings.Contains(text, "127.0.0.1") || !strings.Contains(text, "18081") || !strings.Contains(text, "8443") {
		t.Fatalf("конфиг \n%s", text)
	}
	if !strings.Contains(text, "xtls-rprx-vision") || !strings.Contains(text, "www.cloudflare.com") {
		t.Fatalf("конфиг \n%s", text)
	}
}

func TestParseBootstrapRejectsShort(t *testing.T) {
	if _, err := ParseBootstrap("UUID=short\nPUBLIC_KEY=\nSHORT_ID=1\n"); err == nil {
		t.Fatal("ждали ошибку")
	}
}
