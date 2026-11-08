package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/apppaths"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/cursorfiles"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/fetchbox"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/listen"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/logx"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/procenv"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/scripts"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/settings"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/singboxcfg"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/sshrun"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/taskplan"
)

type Failure struct {
	Step    string
	Message string
}

func (f Failure) Error() string { return f.Step + ": " + f.Message }

type Runner func(ctx context.Context, name string, args ...string) (string, error)

type SSHFunc func(ctx context.Context, creds settings.Credentials, script string) (string, error)

type DownloadFunc func(ctx context.Context, dest string) error

type Engine struct {
	Paths     apppaths.Paths
	Now       func() time.Time
	Sleep     func(ctx context.Context, d time.Duration) error
	Run       Runner
	SSH       SSHFunc
	Download  DownloadFunc
	Bootstrap string
	Teardown  string
	Schtasks  string
	Netstat   string
}

func Exec(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = procenv.Strip(os.Environ())
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return string(out), fmt.Errorf("%s", text)
	}
	return string(out), nil
}

func (e *Engine) Apply(ctx context.Context) error {
	creds, err := e.readCreds()
	if err != nil {
		return e.fail(settings.Credentials{}, "настройки", err)
	}
	if err := e.ensureBinary(ctx, creds); err != nil {
		return err
	}
	e.command(creds, "ssh "+creds.User+"@"+creds.IP+" bash -s")
	stdout, err := e.ssh(ctx, creds, e.Bootstrap)
	if err != nil {
		return e.fail(creds, "ssh", err)
	}
	material, err := singboxcfg.ParseBootstrap(stdout)
	if err != nil {
		return e.fail(creds, "ssh", err)
	}
	client, err := singboxcfg.Client(creds.IP, material)
	if err != nil {
		return e.fail(creds, "клиент", err)
	}
	e.command(creds, "write "+e.Paths.Display(e.Paths.ClientConfig()))
	if err := writeFile(e.Paths.ClientConfig(), []byte(client), 0o600); err != nil {
		return e.fail(creds, "клиент", err)
	}
	if err := e.installDaemon(ctx, creds); err != nil {
		return err
	}
	e.command(creds, "cursor settings "+e.Paths.Display(e.Paths.CursorSettings()))
	if err := rewrite(e.Paths.CursorSettings(), cursorfiles.ApplySettings); err != nil {
		return e.fail(creds, "cursor", err)
	}
	e.command(creds, "cursor argv "+e.Paths.Display(e.Paths.CursorArgv()))
	if err := rewrite(e.Paths.CursorArgv(), cursorfiles.ApplyArgv); err != nil {
		return e.fail(creds, "cursor", err)
	}
	return e.waitListen(ctx, creds)
}

func (e *Engine) Remove(ctx context.Context, alsoVPS bool) error {
	creds, credErr := e.readCreds()
	e.command(creds, "schtasks /End /Delete "+taskplan.TaskName())
	if err := e.stopTask(ctx); err != nil {
		return e.fail(creds, "демон", err)
	}
	e.command(creds, "delete "+e.Paths.Display(e.Paths.ClientConfig()))
	if err := os.Remove(e.Paths.ClientConfig()); err != nil && !os.IsNotExist(err) {
		return e.fail(creds, "клиент", err)
	}
	e.command(creds, "cursor remove settings")
	if err := rewriteIfExists(e.Paths.CursorSettings(), cursorfiles.RemoveSettings); err != nil {
		return e.fail(creds, "cursor", err)
	}
	e.command(creds, "cursor remove argv")
	if err := rewriteIfExists(e.Paths.CursorArgv(), cursorfiles.RemoveArgv); err != nil {
		return e.fail(creds, "cursor", err)
	}
	if !alsoVPS {
		return nil
	}
	if credErr != nil {
		return e.fail(settings.Credentials{}, "настройки", credErr)
	}
	e.command(creds, "ssh "+creds.User+"@"+creds.IP+" bash -s teardown")
	if _, err := e.ssh(ctx, creds, e.Teardown); err != nil {
		return e.fail(creds, "ssh", err)
	}
	return nil
}

func (e *Engine) readCreds() (settings.Credentials, error) {
	raw, err := os.ReadFile(e.Paths.Settings())
	if err != nil {
		return settings.Credentials{}, fmt.Errorf("сначала сохрани настройки")
	}
	creds, ok, err := settings.Read(string(raw))
	if err != nil {
		return settings.Credentials{}, err
	}
	if !ok {
		return settings.Credentials{}, fmt.Errorf("сначала сохрани настройки")
	}
	if err := settings.Validate(creds); err != nil {
		return settings.Credentials{}, err
	}
	return creds, nil
}

func (e *Engine) ensureBinary(ctx context.Context, creds settings.Credentials) error {
	if _, err := os.Stat(e.Paths.SingBox()); err == nil {
		return nil
	}
	url, err := fetchbox.OfficialURL(goArch())
	if err != nil {
		return e.fail(creds, "sing-box", err)
	}
	e.command(creds, "download "+url)
	if err := os.MkdirAll(parent(e.Paths.SingBox()), 0o755); err != nil {
		return e.fail(creds, "sing-box", err)
	}
	download := e.Download
	if download == nil {
		download = func(ctx context.Context, dest string) error {
			return fetchbox.Install(ctx, nil, url, dest)
		}
	}
	if err := download(ctx, e.Paths.SingBox()); err != nil {
		return e.fail(creds, "sing-box", err)
	}
	return nil
}

func (e *Engine) installDaemon(ctx context.Context, creds settings.Credentials) error {
	runner, err := taskplan.RunnerCmd(e.Paths.SingBox(), e.Paths.ClientConfig(), e.Paths.SingBoxLog(), e.Paths.SingBoxErr())
	if err != nil {
		return e.fail(creds, "демон", err)
	}
	xml, err := taskplan.XML(e.Paths.RunnerCmd())
	if err != nil {
		return e.fail(creds, "демон", err)
	}
	if err := writeFile(e.Paths.RunnerCmd(), []byte(runner), 0o700); err != nil {
		return e.fail(creds, "демон", err)
	}
	if err := writeFile(e.Paths.TaskXML(), taskplan.Encode(xml), 0o600); err != nil {
		return e.fail(creds, "демон", err)
	}
	e.command(creds, "schtasks /Create /TN "+taskplan.TaskName())
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		if err := e.startTask(ctx); err != nil {
			last = err
			continue
		}
		return nil
	}
	return e.fail(creds, "демон", last)
}

func (e *Engine) startTask(ctx context.Context) error {
	name := taskplan.TaskName()
	bin := e.schtasksPath()
	_, _ = e.run(ctx, bin, "/End", "/TN", name)
	_, _ = e.run(ctx, bin, "/Delete", "/TN", name, "/F")
	if _, err := e.run(ctx, bin, "/Create", "/TN", name, "/XML", e.Paths.TaskXML(), "/F"); err != nil {
		return err
	}
	if _, err := e.run(ctx, bin, "/Run", "/TN", name); err != nil {
		return err
	}
	return nil
}

func (e *Engine) stopTask(ctx context.Context) error {
	name := taskplan.TaskName()
	bin := e.schtasksPath()
	_, _ = e.run(ctx, bin, "/End", "/TN", name)
	if _, err := e.run(ctx, bin, "/Delete", "/TN", name, "/F"); err != nil {
		text := err.Error()
		if strings.Contains(strings.ToLower(text), "cannot find") || strings.Contains(text, "не удается найти") {
			return nil
		}
		return err
	}
	return nil
}

func (e *Engine) waitListen(ctx context.Context, creds settings.Credentials) error {
	e.command(creds, "netstat -ano -p tcp")
	deadline := e.now().Add(10 * time.Second)
	var last string
	for {
		out, err := e.run(ctx, e.netstatPath(), "-ano", "-p", "tcp")
		last = out
		switch listen.Classify(out) {
		case listen.Ready:
			return nil
		case listen.Exposed:
			return e.fail(creds, "проверка", fmt.Errorf("порт 18081 слушает не только 127.0.0.1: %s", oneLine(out)))
		}
		if err != nil && out == "" {
			last = err.Error()
		}
		if !e.now().Before(deadline) {
			return e.fail(creds, "проверка", fmt.Errorf("не слушает 127.0.0.1:18081: %s", oneLine(last)))
		}
		if err := e.sleep(ctx, 300*time.Millisecond); err != nil {
			return e.fail(creds, "проверка", err)
		}
	}
}

func (e *Engine) ssh(ctx context.Context, creds settings.Credentials, script string) (string, error) {
	script = scripts.Unix(script)
	if e.SSH != nil {
		return e.SSH(ctx, creds, script)
	}
	callback := sshrun.Remember(e.Paths.KnownHosts())
	return sshrun.Run(ctx, creds.IP+":22", creds.User, creds.Password, script, callback)
}

func (e *Engine) run(ctx context.Context, name string, args ...string) (string, error) {
	if e.Run != nil {
		return e.Run(ctx, name, args...)
	}
	return Exec(ctx, name, args...)
}

func (e *Engine) command(creds settings.Credentials, command string) {
	_ = logx.Append(e.Paths.CommandLog(), logx.CommandLine(e.now(), command, creds.Secrets()))
}

func (e *Engine) fail(creds settings.Credentials, step string, err error) error {
	msg := logx.Mask(err.Error(), creds.Secrets())
	_ = logx.Append(e.Paths.CommandLog(), logx.ErrorLine(e.now(), step, msg, nil))
	return Failure{Step: step, Message: msg}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) sleep(ctx context.Context, d time.Duration) error {
	if e.Sleep != nil {
		return e.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (e *Engine) schtasksPath() string {
	if e.Schtasks != "" {
		return e.Schtasks
	}
	return "schtasks"
}

func (e *Engine) netstatPath() string {
	if e.Netstat != "" {
		return e.Netstat
	}
	return "netstat"
}

func goArch() string {
	if v := os.Getenv("CURSOR_FAST_PROXY_ARCH"); v != "" {
		return v
	}
	return arch()
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(parent(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func parent(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i <= 0 {
		return "."
	}
	return path[:i]
}

func rewrite(path string, edit func(string) (string, error)) error {
	raw, err := os.ReadFile(path)
	source := ""
	if err == nil {
		source = string(raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	next, err := edit(source)
	if err != nil {
		return err
	}
	return writeFile(path, []byte(next), 0o600)
}

func rewriteIfExists(path string, edit func(string) (string, error)) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	next, err := edit(string(raw))
	if err != nil {
		return err
	}
	return writeFile(path, []byte(next), 0o600)
}

func oneLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "пусто"
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if len(text) > 240 {
		text = text[:240]
	}
	return text
}
