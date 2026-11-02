package singboxcfg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
)

type Bootstrap struct {
	UUID      string
	PublicKey string
	ShortID   string
}

func ParseBootstrap(output string) (Bootstrap, error) {
	values := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		name, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		values[name] = value
	}
	out := Bootstrap{
		UUID:      values["UUID"],
		PublicKey: values["PUBLIC_KEY"],
		ShortID:   values["SHORT_ID"],
	}
	if len(out.UUID) < 32 || out.PublicKey == "" || len(out.ShortID) < 8 {
		return Bootstrap{}, fmt.Errorf("bootstrap не вернул UUID, PUBLIC_KEY и SHORT_ID")
	}
	return out, nil
}

func Client(creds settings.Credentials, keys Bootstrap) ([]byte, error) {
	listen := fmt.Sprintf("%s:%d", canon.ListenHost, canon.ListenPort)
	if listen != "127.0.0.1:18081" {
		return nil, fmt.Errorf("listen только 127.0.0.1:18081")
	}
	doc := map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{
			map[string]any{
				"type":        "mixed",
				"tag":         "cursor-in",
				"listen":      canon.ListenHost,
				"listen_port": canon.ListenPort,
			},
		},
		"outbounds": []any{
			map[string]any{
				"type":        "vless",
				"tag":         "vps",
				"server":      creds.IP,
				"server_port": canon.ServerPort,
				"uuid":        keys.UUID,
				"flow":        "xtls-rprx-vision",
				"tls": map[string]any{
					"enabled":     true,
					"server_name": canon.SNI,
					"utls": map[string]any{
						"enabled":     true,
						"fingerprint": "chrome",
					},
					"reality": map[string]any{
						"enabled":    true,
						"public_key": keys.PublicKey,
						"short_id":   keys.ShortID,
					},
				},
			},
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"inbound":  []string{"cursor-in"},
					"outbound": "vps",
				},
			},
			"final": "direct",
		},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	text := string(raw)
	if strings.Contains(text, `"tun"`) {
		return nil, fmt.Errorf("клиентский конфиг без tun")
	}
	if !strings.Contains(text, `"listen": "127.0.0.1"`) {
		return nil, fmt.Errorf("listen не закреплён за loopback")
	}
	return append(raw, '\n'), nil
}
