package settings

import (
	"fmt"
	"net"
	"strings"
	"unicode"
)

const (
	begin = "# >>> cursor-fast-proxy >>>"
	end   = "# <<< cursor-fast-proxy <<<"
)

type Credentials struct {
	IP       string
	User     string
	Password string
}

type Error struct{ Message string }

func (e Error) Error() string { return e.Message }

func Validate(creds Credentials) error {
	if strings.TrimSpace(creds.IP) == "" || strings.TrimSpace(creds.User) == "" || creds.Password == "" {
		return Error{Message: "заполни ip, user и password"}
	}
	if strings.ContainsAny(creds.IP, " \t\r\n") || net.ParseIP(creds.IP) == nil {
		return Error{Message: "ip должен быть адресом"}
	}
	if strings.ContainsAny(creds.User, " \t\r\n") {
		return Error{Message: "user без пробелов"}
	}
	if strings.ContainsAny(creds.Password, "\r\n") {
		return Error{Message: "password без перевода строки"}
	}
	return nil
}

func Write(creds Credentials) (string, error) {
	if err := Validate(creds); err != nil {
		return "", err
	}
	return begin + "\n" +
		"CURSOR_FAST_PROXY_IP=" + shellQuote(creds.IP) + "\n" +
		"CURSOR_FAST_PROXY_USER=" + shellQuote(creds.User) + "\n" +
		"CURSOR_FAST_PROXY_PASS=" + shellQuote(creds.Password) + "\n" +
		end + "\n", nil
}

func Read(text string) (Credentials, bool, error) {
	start := strings.Index(text, begin)
	if start < 0 {
		return Credentials{}, false, nil
	}
	rest := text[start+len(begin):]
	stop := strings.Index(rest, end)
	if stop < 0 {
		return Credentials{}, false, Error{Message: "блок cursor-fast-proxy не закрыт"}
	}
	values := map[string]string{}
	for _, line := range strings.Split(rest[:stop], "\n") {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(line[len("export "):])
		}
		name, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value, err := unescape(strings.TrimSpace(raw))
		if err != nil {
			return Credentials{}, false, err
		}
		values[name] = value
	}
	creds := Credentials{
		IP:       values["CURSOR_FAST_PROXY_IP"],
		User:     values["CURSOR_FAST_PROXY_USER"],
		Password: values["CURSOR_FAST_PROXY_PASS"],
	}
	if err := Validate(creds); err != nil {
		return Credentials{}, false, err
	}
	return creds, true, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func unescape(raw string) (string, error) {
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		body := raw[1 : len(raw)-1]
		return strings.ReplaceAll(body, `'"'"'`, "'"), nil
	}
	if strings.ContainsAny(raw, " \t") {
		return "", Error{Message: "значение без кавычек содержит пробел"}
	}
	return raw, nil
}

func FormatReadError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%s", err)
}
