package taskplan

import (
	"strings"
	"testing"
)

func TestXMLStartsAtLogonAndRestarts(t *testing.T) {
	xml, err := XML(`C:\Users\me\AppData\Local\CursorFastProxy\run-singbox.cmd`)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"Cursor fast proxy sing-box",
		"<LogonTrigger>",
		"<Interval>PT10S</Interval>",
		"<ExecutionTimeLimit>PT0S</ExecutionTimeLimit>",
		"<RunLevel>LeastPrivilege</RunLevel>",
		`C:\Users\me\AppData\Local\CursorFastProxy\run-singbox.cmd`,
	} {
		if !strings.Contains(xml, needle) {
			t.Fatalf("нет %s", needle)
		}
	}
	if strings.Contains(xml, "HighestAvailable") || strings.Contains(xml, "tun") {
		t.Fatal("задача повысила права или упомянула tun")
	}
}

func TestRunnerCmdQuotesPaths(t *testing.T) {
	got, err := RunnerCmd(
		`C:\Program Files\sing-box.exe`,
		`C:\Users\me\cursor-egress\sing-box-client.json`,
		`C:\Users\me\AppData\Local\CursorFastProxy\sing-box.log`,
		`C:\Users\me\AppData\Local\CursorFastProxy\sing-box.err`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"C:\Program Files\sing-box.exe" run -c "C:\Users\me\cursor-egress\sing-box-client.json"`) {
		t.Fatalf("%s", got)
	}
}

func TestRejectsMetacharacters(t *testing.T) {
	if _, err := RunnerCmd(`C:\a&b\sing-box.exe`, `C:\c.json`, `C:\l`, `C:\e`); err == nil {
		t.Fatal("амперсанд")
	}
	if _, err := XML(`C:\bad&path.cmd`); err == nil {
		t.Fatal("xml")
	}
}

func TestTaskName(t *testing.T) {
	if TaskName() != "ru.local.cursor-singbox" {
		t.Fatal(TaskName())
	}
}
