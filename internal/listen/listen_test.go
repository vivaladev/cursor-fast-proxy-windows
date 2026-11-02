package listen

import "testing"

func TestReadyOnLoopbackNetstat(t *testing.T) {
	sample := "  TCP    127.0.0.1:18081        0.0.0.0:0              LISTENING       1408\n"
	if Classify(sample) != Ready {
		t.Fatal(Classify(sample))
	}
}

func TestWaitingWhenSilent(t *testing.T) {
	if Classify("  TCP    127.0.0.1:443        0.0.0.0:0              LISTENING       4\n") != Waiting {
		t.Fatal("чужой порт")
	}
}

func TestExposedRejectsWildcard(t *testing.T) {
	for _, line := range []string{
		"  TCP    0.0.0.0:18081         0.0.0.0:0              LISTENING       9\n",
		"  TCP    *:18081               *:*                    LISTEN\n",
		"  TCP    [::]:18081            [::]:0                 LISTENING       9\n",
	} {
		if Classify(line) != Exposed {
			t.Fatalf("%q -> %v", line, Classify(line))
		}
	}
}

func TestIPv6LocalhostAloneIsNotReady(t *testing.T) {
	line := "  TCP    [::1]:18081            [::]:0                 LISTENING       9\n"
	if Classify(line) != Waiting {
		t.Fatal(Classify(line))
	}
}
