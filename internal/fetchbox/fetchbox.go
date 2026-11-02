package fetchbox

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

func AssetURL(version, arch string) (string, error) {
	switch arch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("sing-box: нужна 64-битная Windows, сейчас %s", arch)
	}
	return fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-windows-%s.zip", version, version, arch), nil
}

func OfficialURL(arch string) (string, error) {
	return AssetURL(canon.SingBoxVersion, arch)
}

func Install(ctx context.Context, client *http.Client, rawURL, dest string) error {
	if client == nil {
		client = &http.Client{Transport: &http.Transport{Proxy: nil}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sing-box: не скачался архив: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sing-box: архив ответил %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 80<<20))
	if err != nil {
		return fmt.Errorf("sing-box: архив оборвался: %w", err)
	}
	return ExtractExe(body, dest)
}

func ExtractExe(body []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("sing-box: это не zip: %w", err)
	}
	var file *zip.File
	for _, item := range reader.File {
		if strings.Contains(item.Name, "..") {
			return fmt.Errorf("sing-box: архив содержит %s", item.Name)
		}
		if filepath.Base(item.Name) == "sing-box.exe" && !item.FileInfo().IsDir() {
			file = item
			break
		}
	}
	if file == nil {
		return fmt.Errorf("sing-box: в архиве нет sing-box.exe")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(src, 80<<20)); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}
