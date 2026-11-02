package cursorfiles

import (
	"strings"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/jsontext"
)

func ApplySettings(source string) (string, error) {
	text := normalize(source)
	var err error
	text, err = jsontext.Set(text, "http.proxy", jsontext.String(canon.ProxyURL))
	if err != nil {
		return "", err
	}
	text, err = jsontext.Set(text, "http.proxySupport", jsontext.String("override"))
	if err != nil {
		return "", err
	}
	text, err = jsontext.Set(text, "cursor.general.disableHttp2", jsontext.Bool(true))
	if err != nil {
		return "", err
	}
	return jsontext.Set(text, "http.noProxy", jsontext.String(canon.NoProxy))
}

func RemoveSettings(source string) (string, error) {
	text := source
	var err error
	for _, key := range canon.SettingsKeys {
		text, err = jsontext.Remove(text, key)
		if err != nil {
			return "", err
		}
	}
	return text, nil
}

func ApplyArgv(source string) (string, error) {
	text := normalize(source)
	var err error
	text, err = jsontext.Set(text, "proxy-server", jsontext.String(canon.ProxyURL))
	if err != nil {
		return "", err
	}
	return jsontext.Set(text, "proxy-bypass-list", jsontext.String(canon.BypassList))
}

func RemoveArgv(source string) (string, error) {
	text := source
	var err error
	for _, key := range canon.ArgvKeys {
		text, err = jsontext.Remove(text, key)
		if err != nil {
			return "", err
		}
	}
	return text, nil
}

func normalize(source string) string {
	if strings.TrimSpace(source) == "" {
		return "{\n}\n"
	}
	return source
}
