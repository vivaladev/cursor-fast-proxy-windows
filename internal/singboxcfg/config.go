package singboxcfg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

type RealityPublic struct {
	UUID      string
	PublicKey string
	ShortID   string
}

func Client(server string, material RealityPublic) (string, error) {
	object := map[string]any{
		"log": map[string]any{"level": "warn", "timestamp": true},
		"inbounds": []any{map[string]any{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      canon.ListenHost,
			"listen_port": canon.ListenPort,
		}},
		"outbounds": []any{map[string]any{
			"type":        "vless",
			"tag":         "vps",
			"server":      server,
			"server_port": canon.ServerPort,
			"uuid":        material.UUID,
			"flow":        "xtls-rprx-vision",
			"tls": map[string]any{
				"enabled":     true,
				"server_name": canon.SNI,
				"utls":        map[string]any{"enabled": true, "fingerprint": "chrome"},
				"reality": map[string]any{
					"enabled":    true,
					"public_key": material.PublicKey,
					"short_id":   material.ShortID,
				},
			},
		}},
	}
	raw, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return "", fmt.Errorf("клиент: не собрался JSON клиента")
	}
	text := string(raw)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if strings.Contains(text, `"tun"`) {
		return "", fmt.Errorf("клиент: клиентский конфиг содержит tun")
	}
	if strings.Contains(text, `"listen": "0.0.0.0"`) || strings.Contains(text, `"listen": "::"`) {
		return "", fmt.Errorf("клиент: inbound слушает не loopback")
	}
	return text, nil
}

func ParseBootstrap(stdout string) (RealityPublic, error) {
	grab := func(name string) (string, error) {
		prefix := name + "="
		for _, line := range strings.Split(stdout, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			if value == "" {
				break
			}
			return value, nil
		}
		return "", fmt.Errorf("ssh: в ответе VPS нет %s", name)
	}
	uuid, err := grab("UUID")
	if err != nil {
		return RealityPublic{}, err
	}
	publicKey, err := grab("PUBLIC_KEY")
	if err != nil {
		return RealityPublic{}, err
	}
	shortID, err := grab("SHORT_ID")
	if err != nil {
		return RealityPublic{}, err
	}
	material := RealityPublic{UUID: uuid, PublicKey: publicKey, ShortID: shortID}
	if len(material.UUID) < 32 || material.PublicKey == "" || len(material.ShortID) < 8 {
		return RealityPublic{}, fmt.Errorf("ssh: ответ VPS не похож на ключи Reality")
	}
	return material, nil
}
