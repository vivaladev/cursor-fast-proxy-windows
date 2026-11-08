package taskplan

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/canon"
)

func XML(command string) (string, error) {
	if err := safePath(command); err != nil {
		return "", err
	}
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Cursor fast proxy sing-box</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT10S</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + xmlEscape(command) + `</Command>
    </Exec>
  </Actions>
</Task>
`, nil
}

// Encode is the task.xml schtasks accepts on a Russian Windows: UTF-16 LE with BOM.
// UTF-8 without a BOM is read as the OEM code page, and the encoding
// declaration then fails at column 40.
func Encode(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, 2+len(units)*2)
	out[0] = 0xFF
	out[1] = 0xFE
	for i, unit := range units {
		out[2+i*2] = byte(unit)
		out[3+i*2] = byte(unit >> 8)
	}
	return out
}

func RunnerCmd(singBox, config, logPath, errPath string) (string, error) {
	for _, path := range []string{singBox, config, logPath, errPath} {
		if err := safePath(path); err != nil {
			return "", err
		}
	}
	return "@echo off\r\n\"" + singBox + "\" run -c \"" + config + "\" >> \"" + logPath + "\" 2>> \"" + errPath + "\"\r\n", nil
}

func safePath(path string) error {
	if path == "" || strings.ContainsAny(path, "\r\n\"&%<>") {
		return fmt.Errorf("демон: небезопасный путь %q", path)
	}
	return nil
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func TaskName() string { return canon.Label }
