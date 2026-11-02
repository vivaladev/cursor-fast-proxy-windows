package listen

import "strings"

type Status int

const (
	Waiting Status = iota
	Ready
	Exposed
)

func Classify(output string) Status {
	var hits []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, ":18081") {
			hits = append(hits, line)
		}
	}
	if len(hits) == 0 {
		return Waiting
	}
	ready := false
	for _, line := range hits {
		if exposed(line) {
			return Exposed
		}
		if strings.Contains(line, "127.0.0.1:18081") {
			ready = true
		}
	}
	if ready {
		return Ready
	}
	return Waiting
}

func exposed(line string) bool {
	return strings.Contains(line, "0.0.0.0:18081") ||
		strings.Contains(line, "*:18081") ||
		strings.Contains(line, "[::]:18081") ||
		strings.Contains(line, "[::0]:18081")
}
