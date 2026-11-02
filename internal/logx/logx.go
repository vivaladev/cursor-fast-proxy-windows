package logx

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Format(date time.Time) string {
	return date.Format("2006-01-02T15:04:05")
}

func Mask(text string, secrets map[string]string) string {
	type pair struct {
		name  string
		value string
	}
	pairs := make([]pair, 0, len(secrets))
	for name, value := range secrets {
		if value == "" {
			continue
		}
		pairs = append(pairs, pair{name, value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if len(pairs[i].value) == len(pairs[j].value) {
			return pairs[i].name < pairs[j].name
		}
		return len(pairs[i].value) > len(pairs[j].value)
	})
	result := text
	for _, item := range pairs {
		result = strings.ReplaceAll(result, item.value, item.name)
	}
	return result
}

func CommandLine(date time.Time, command string, secrets map[string]string) string {
	return Format(date) + "-" + Mask(command, secrets)
}

func ErrorLine(date time.Time, step, message string, secrets map[string]string) string {
	return Format(date) + "-ERROR " + step + ": " + Mask(message, secrets)
}

func Append(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}
