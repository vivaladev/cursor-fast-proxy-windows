package procenv

import "strings"

var proxyNames = map[string]struct{}{
	"HTTP_PROXY": {}, "HTTPS_PROXY": {}, "ALL_PROXY": {}, "NO_PROXY": {},
	"http_proxy": {}, "https_proxy": {}, "all_proxy": {}, "no_proxy": {},
}

func Strip(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, item := range environ {
		name, _, ok := strings.Cut(item, "=")
		if !ok {
			out = append(out, item)
			continue
		}
		if _, drop := proxyNames[name]; drop {
			continue
		}
		out = append(out, item)
	}
	return out
}
