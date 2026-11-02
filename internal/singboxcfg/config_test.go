package singboxcfg

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

func sample() RealityPublic {
	return RealityPublic{
		UUID:      "11111111-1111-4111-8111-111111111111",
		PublicKey: "PUBLICKEYPUBLICKEYPUBLICKEY12abcd",
		ShortID:   "abcdef12",
	}
}

func TestClientLoopbackWithoutTunOrPrivateKey(t *testing.T) {
	text, err := Client("203.0.113.10", sample())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(text)) {
		t.Fatalf("не json: %s", text)
	}
	for _, needle := range []string{
		`"listen": "127.0.0.1"`,
		`"listen_port": 18081`,
		`"server": "203.0.113.10"`,
		`"server_port": 8443`,
		`"flow": "xtls-rprx-vision"`,
		`"server_name": "www.cloudflare.com"`,
		`"public_key": "PUBLICKEYPUBLICKEYPUBLICKEY12abcd"`,
		`"short_id": "abcdef12"`,
		`"fingerprint": "chrome"`,
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("нет %s в %s", needle, text)
		}
	}
	if strings.Contains(text, "tun") || strings.Contains(text, "PRIVATE") || strings.Contains(text, "0.0.0.0") {
		t.Fatalf("лишнее в клиенте: %s", text)
	}
	if strings.Contains(text, canon.ListenHost) && strings.Contains(text, `"listen": "0.0.0.0"`) {
		t.Fatal("слушает все интерфейсы")
	}
}

func TestParseBootstrap(t *testing.T) {
	stdout := "progress\nUUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=PUBLICKEYPUBLICKEYPUBLICKEY12abcd\nSHORT_ID=abcdef1234567890\n"
	got, err := ParseBootstrap(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "11111111-1111-4111-8111-111111111111" || got.ShortID != "abcdef1234567890" {
		t.Fatalf("%+v", got)
	}
}

func TestParseBootstrapRejectsIncomplete(t *testing.T) {
	if _, err := ParseBootstrap("UUID=11111111-1111-4111-8111-111111111111\n"); err == nil {
		t.Fatal("нет PUBLIC_KEY")
	}
	if _, err := ParseBootstrap("UUID=short\nPUBLIC_KEY=k\nSHORT_ID=abcdef12\n"); err == nil {
		t.Fatal("короткий uuid")
	}
	if _, err := ParseBootstrap("UUID=11111111-1111-4111-8111-111111111111\nPUBLIC_KEY=k\nSHORT_ID=abc\n"); err == nil {
		t.Fatal("короткий short id")
	}
}
