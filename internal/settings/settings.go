package settings

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	Begin = "# >>> cursor-fast-proxy >>>"
	End   = "# <<< cursor-fast-proxy <<<"
)

type Credentials struct {
	IP       string
	User     string
	Password string
}

func (c Credentials) Secrets() map[string]string {
	return map[string]string{
		"CURSOR_FAST_PROXY_IP":   c.IP,
		"CURSOR_FAST_PROXY_USER": c.User,
		"CURSOR_FAST_PROXY_PASS": c.Password,
	}
}

type Error struct{ Message string }

func (e Error) Error() string { return e.Message }

func Validate(c Credentials) error {
	if err := require(c.IP, "ip", false); err != nil {
		return err
	}
	if err := require(c.User, "user", false); err != nil {
		return err
	}
	return require(c.Password, "password", true)
}

func require(value, name string, allowSpaces bool) error {
	if value == "" || strings.Contains(value, "\n") || strings.Contains(value, "\r") {
		return Error{Message: "пустое поле " + name}
	}
	if !allowSpaces {
		for _, r := range value {
			if unicode.IsSpace(r) {
				return Error{Message: "пустое поле " + name}
			}
		}
	}
	return nil
}

func Write(c Credentials) (string, error) {
	if err := Validate(c); err != nil {
		return "", err
	}
	lines := []string{
		Begin,
		"export CURSOR_FAST_PROXY_IP=" + shellQuote(c.IP),
		"export CURSOR_FAST_PROXY_USER=" + shellQuote(c.User),
		"export CURSOR_FAST_PROXY_PASS=" + shellQuote(c.Password),
		End,
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func Read(contents string) (Credentials, bool, error) {
	lines := splitLines(contents)
	start := indexOf(lines, Begin)
	stop := indexOf(lines, End)
	if start < 0 {
		if stop >= 0 {
			return Credentials{}, false, Error{Message: "блок cursor-fast-proxy в настройках оборван"}
		}
		return Credentials{}, false, nil
	}
	if stop < 0 || stop <= start {
		return Credentials{}, false, Error{Message: "блок cursor-fast-proxy в настройках оборван"}
	}
	body := lines[start+1 : stop]
	ip, okIP := value(body, "CURSOR_FAST_PROXY_IP")
	user, okUser := value(body, "CURSOR_FAST_PROXY_USER")
	pass, okPass := value(body, "CURSOR_FAST_PROXY_PASS")
	if !okIP || !okUser || !okPass {
		return Credentials{}, false, nil
	}
	return Credentials{IP: ip, User: user, Password: pass}, true, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func value(body []string, name string) (string, bool) {
	prefix := "export " + name + "="
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			raw := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			text, ok := unescape(raw)
			if !ok {
				return "", false
			}
			return text, true
		}
	}
	return "", false
}

func unescape(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", false
	}
	inner := raw[1 : len(raw)-1]
	return strings.ReplaceAll(inner, `'\''`, "'"), true
}

func splitLines(contents string) []string {
	text := contents
	if strings.HasSuffix(text, "\n") {
		text = text[:len(text)-1]
	}
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func indexOf(lines []string, target string) int {
	for i, line := range lines {
		if line == target {
			return i
		}
	}
	return -1
}

func FormatReadError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}
