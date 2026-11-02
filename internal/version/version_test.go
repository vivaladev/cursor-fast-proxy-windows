package version

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCurrentMatchesFile(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("нет пути к тесту")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != Current {
		t.Fatalf("VERSION %q, константа %q", strings.TrimSpace(string(raw)), Current)
	}
}
