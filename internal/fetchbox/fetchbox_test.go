package fetchbox

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

func TestAssetURL(t *testing.T) {
	got, err := OfficialURL("amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/SagerNet/sing-box/releases/download/v" + canon.SingBoxVersion + "/sing-box-" + canon.SingBoxVersion + "-windows-amd64.zip"
	if got != want {
		t.Fatalf("%s", got)
	}
	if _, err := OfficialURL("386"); err == nil {
		t.Fatal("32-bit")
	}
}

func TestInstallExtractsExeAndRejectsZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entry, err := zw.Create("sing-box-" + canon.SingBoxVersion + "-windows-amd64/sing-box.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("binary")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	good := buf.Bytes()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(good)
	}))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "bin", "sing-box.exe")
	if err := Install(context.Background(), server.Client(), server.URL, dest); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "binary" {
		t.Fatalf("%q", raw)
	}

	slip := bytes.Buffer{}
	zw = zip.NewWriter(&slip)
	bad, err := zw.Create("../sing-box.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = bad.Write([]byte("nope"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ExtractExe(slip.Bytes(), filepath.Join(t.TempDir(), "sing-box.exe")); err == nil || !strings.Contains(err.Error(), "..") {
		t.Fatalf("zip slip: %v", err)
	}
}
