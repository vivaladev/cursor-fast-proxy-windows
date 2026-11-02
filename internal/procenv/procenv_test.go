package procenv

import "testing"

func TestStripRemovesProxy(t *testing.T) {
	got := Strip([]string{"PATH=/usr/bin", "HTTP_PROXY=http://127.0.0.1:18081", "http_proxy=http://127.0.0.1:18081", "ALL_PROXY=socks5://127.0.0.1:18081", "HOME=/Users/me"})
	if len(got) != 2 || got[0] != "PATH=/usr/bin" || got[1] != "HOME=/Users/me" {
		t.Fatalf("%v", got)
	}
}
