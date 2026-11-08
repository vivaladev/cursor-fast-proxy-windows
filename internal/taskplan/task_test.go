package taskplan

import (
	"strings"
	"testing"
	"unicode/utf16"
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
	if !strings.HasPrefix(xml, `<?xml version="1.0" encoding="UTF-16"?>`) || strings.Contains(xml, "UTF-8") {
		t.Fatalf("объявление кодировки %q", xml[:80])
	}
	raw := Encode(xml)
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xFE {
		t.Fatalf("BOM %x", raw[:2])
	}
	units := make([]uint16, (len(raw)-2)/2)
	for i := range units {
		units[i] = uint16(raw[2+i*2]) | uint16(raw[3+i*2])<<8
	}
	if string(utf16.Decode(units)) != xml {
		t.Fatal("UTF-16 LE не совпал с текстом задачи")
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
	if !strings.Contains(got, "\r\n") || strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Fatal("cmd потерял CRLF")
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
