//go:build windows

package main

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/workflow"
	"golang.org/x/sys/windows"
)

const (
	wsCaption      = 0x00C00000
	wsSysMenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsVisible      = 0x10000000
	wsChild        = 0x40000000
	wsTabStop      = 0x00010000
	wsVScroll      = 0x00200000
	esAutohscroll  = 0x0080
	esPassword     = 0x0020
	esMultiline    = 0x0004
	esReadonly     = 0x0800
	esAutovscroll  = 0x0040
	wmDestroy      = 0x0002
	wmClose        = 0x0010
	wmCommand      = 0x0111
	wmSetFont      = 0x0030
	emSetLimitText = 0x00C5
	swShow         = 5
	swHide         = 0
	wmStatus       = 0x8000 + 1

	idApply    = 101
	idRemove   = 102
	idSettings = 103
	idLogs     = 104
	idSave     = 105
	idBack     = 106
	idYes      = 107
	idYesVPS   = 108
	idIP       = 201
	idUser     = 202
	idPass     = 203
	idStatus   = 301
	idLogBox   = 302
	idHint     = 303
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procShowWindow         = user32.NewProc("ShowWindow")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW  = user32.NewProc("GetWindowTextLengthW")
	procMessageBoxW        = user32.NewProc("MessageBoxW")
	procLoadCursorW        = user32.NewProc("LoadCursorW")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procEnableWindow       = user32.NewProc("EnableWindow")
	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procCreateFontW        = gdi32.NewProc("CreateFontW")

	windowCallback = windows.NewCallback(windowProc)
	state          *desk
	statusMu       sync.Mutex
	pendingStatus  string
	pendingEnable  bool
)

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type point struct{ X, Y int32 }

type message struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type desk struct {
	engine  *workflow.Engine
	hwnd    uintptr
	font    uintptr
	status  uintptr
	hint    uintptr
	logBox  uintptr
	buttons []uintptr
	main    []uintptr
	fields  []uintptr
	confirm []uintptr
	back    uintptr
	screen  string
}

func runWindow(engine *workflow.Engine) error {
	procSetProcessDPIAware.Call()
	instance, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, 32512)
	className := utf16("CursorFastProxyWindow")
	class := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   windowCallback,
		HInstance:     instance,
		HCursor:       cursor,
		HbrBackground: 6,
		LpszClassName: className,
	}
	if atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return fmt.Errorf("окно: %v", err)
	}
	title := utf16(windowTitle())
	state = &desk{engine: engine, screen: "main"}
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsCaption|wsSysMenu|wsMinimizeBox,
		120, 80, 480, 640,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("окно не создалось: %v", err)
	}
	state.hwnd = hwnd
	state.font = makeFont("Segoe UI", -18)
	state.build(instance)
	state.show("main")
	setText(state.status, "Впиши ip, user и password, сохрани и нажми «Вжух и прокси».")
	procShowWindow.Call(hwnd, swShow)
	var msg message
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return nil
}

func (d *desk) build(instance uintptr) {
	d.hint = d.child("STATIC", "Cursor ходит в облако через 127.0.0.1:18081. Остальной трафик Windows не трогается.", 0, 16, 16, 440, 48, idHint, instance)
	d.status = d.child("EDIT", "", wsVScroll|esMultiline|esReadonly|esAutovscroll, 16, 72, 440, 120, idStatus, instance)
	limit(d.status)
	d.main = []uintptr{
		d.child("BUTTON", "Вжух и прокси", 0, 16, 210, 440, 40, idApply, instance),
		d.child("BUTTON", "Удалить настройки прокси", 0, 16, 258, 440, 40, idRemove, instance),
		d.child("BUTTON", "Настройки", 0, 16, 306, 440, 40, idSettings, instance),
		d.child("BUTTON", "Логи", 0, 16, 354, 440, 40, idLogs, instance),
	}
	d.fields = []uintptr{
		d.child("STATIC", "ip", 0, 16, 210, 80, 22, 0, instance),
		d.child("EDIT", "", esAutohscroll, 110, 206, 346, 28, idIP, instance),
		d.child("STATIC", "user", 0, 16, 248, 80, 22, 0, instance),
		d.child("EDIT", "", esAutohscroll, 110, 244, 346, 28, idUser, instance),
		d.child("STATIC", "password", 0, 16, 286, 80, 22, 0, instance),
		d.child("EDIT", "", esAutohscroll|esPassword, 110, 282, 346, 28, idPass, instance),
		d.child("BUTTON", "Сохранить", 0, 16, 330, 210, 40, idSave, instance),
	}
	d.confirm = []uintptr{
		d.child("BUTTON", "Да", 0, 16, 210, 440, 40, idYes, instance),
		d.child("BUTTON", "Да, и на vps", 0, 16, 258, 440, 40, idYesVPS, instance),
	}
	d.logBox = d.child("EDIT", "", wsVScroll|esMultiline|esReadonly|esAutovscroll, 16, 206, 440, 250, idLogBox, instance)
	limit(d.logBox)
	d.back = d.child("BUTTON", "Назад", 0, 16, 470, 440, 40, idBack, instance)
	d.buttons = append(append(append([]uintptr{}, d.main...), d.fields...), d.confirm...)
	d.buttons = append(d.buttons, d.back)
}

func (d *desk) child(class, text string, extra uintptr, x, y, w, h int, id int, instance uintptr) uintptr {
	style := uintptr(wsChild | wsVisible | wsTabStop)
	style |= extra
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16(class))),
		uintptr(unsafe.Pointer(utf16(text))),
		style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		d.hwnd,
		uintptr(id),
		instance,
		0,
	)
	if d.font != 0 && hwnd != 0 {
		procSendMessageW.Call(hwnd, wmSetFont, d.font, 1)
	}
	return hwnd
}

func (d *desk) show(screen string) {
	d.screen = screen
	for _, hwnd := range append(append([]uintptr{}, d.main...), d.fields...) {
		show(hwnd, false)
	}
	for _, hwnd := range d.confirm {
		show(hwnd, false)
	}
	show(d.logBox, false)
	show(d.back, false)
	switch screen {
	case "main":
		for _, hwnd := range d.main {
			show(hwnd, true)
		}
		setText(d.hint, "Cursor ходит в облако через 127.0.0.1:18081. Остальной трафик Windows не трогается.")
	case "settings":
		for _, hwnd := range d.fields {
			show(hwnd, true)
		}
		show(d.back, true)
		creds, err := loadSettings(d.engine)
		if err != nil {
			setText(d.status, err.Error())
		}
		setByID(idIP, creds.IP)
		setByID(idUser, creds.User)
		setByID(idPass, creds.Password)
		setText(d.hint, "Сохранить пишет блок в ~\\.cursor-fast-proxy\\settings.env и не ставит системный прокси.")
	case "confirm":
		for _, hwnd := range d.confirm {
			show(hwnd, true)
		}
		show(d.back, true)
		setText(d.hint, "«Да» убирает локальный прокси. «Да, и на vps» ещё удаляет сервис sing-box-cursor.")
	case "logs":
		show(d.logBox, true)
		show(d.back, true)
		setText(d.logBox, readLog(d.engine))
		setText(d.hint, "В журнале ip, user и password заменены именами переменных.")
	}
}

func (d *desk) onCommand(id uintptr) {
	switch id {
	case idApply:
		d.busy(func() error { return applyProxy(d.engine) }, "Готово. Полностью перезапусти Cursor, не Reload Window.")
	case idRemove:
		d.show("confirm")
	case idSettings:
		d.show("settings")
	case idLogs:
		d.show("logs")
	case idBack:
		d.show("main")
	case idSave:
		err := saveSettings(d.engine, textByID(idIP), textByID(idUser), textByID(idPass))
		if err != nil {
			setText(d.status, err.Error())
			return
		}
		setText(d.status, "Сохранено. Открытые терминалы сами не подхватят файл: приложению он не нужен.")
		d.show("main")
	case idYes:
		d.busy(func() error { return removeProxy(d.engine, false) }, "Прокси убран из Cursor. Полностью перезапусти Cursor.")
	case idYesVPS:
		d.busy(func() error { return removeProxy(d.engine, true) }, "Локальный прокси и сервис на VPS удалены. Полностью перезапусти Cursor.")
	}
}

func (d *desk) busy(work func() error, okText string) {
	d.setEnabled(false)
	setText(d.status, "Секунду...")
	go func() {
		err := work()
		if err != nil {
			postStatus(err.Error(), true)
			return
		}
		postStatus(okText, true)
	}()
}

func (d *desk) setEnabled(enabled bool) {
	flag := uintptr(0)
	if enabled {
		flag = 1
	}
	for _, hwnd := range d.buttons {
		procEnableWindow.Call(hwnd, flag)
	}
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCommand:
		if state != nil && (wParam>>16) == 0 {
			state.onCommand(wParam & 0xFFFF)
		}
		return 0
	case wmStatus:
		statusMu.Lock()
		text := pendingStatus
		enable := pendingEnable
		statusMu.Unlock()
		if state != nil {
			setText(state.status, text)
			if enable {
				state.setEnabled(true)
				state.show("main")
			}
		}
		return 0
	case wmClose, wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func postStatus(text string, enable bool) {
	statusMu.Lock()
	pendingStatus = text
	pendingEnable = enable
	statusMu.Unlock()
	if state != nil {
		procPostMessageW.Call(state.hwnd, wmStatus, 0, 0)
	}
}

func setText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16(text))))
}

func setByID(id int, text string) {
	setText(find(id), text)
}

func textByID(id int) string {
	return getText(find(id))
}

func find(id int) uintptr {
	hwnd, _, _ := user32.NewProc("GetDlgItem").Call(state.hwnd, uintptr(id))
	return hwnd
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLenW.Call(hwnd)
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return windows.UTF16ToString(buf)
}

func show(hwnd uintptr, visible bool) {
	cmd := uintptr(swHide)
	if visible {
		cmd = swShow
	}
	procShowWindow.Call(hwnd, cmd)
}

func limit(hwnd uintptr) {
	procSendMessageW.Call(hwnd, emSetLimitText, 1<<20, 0)
}

func makeFont(name string, height int) uintptr {
	font, _, _ := procCreateFontW.Call(
		uintptr(height),
		0, 0, 0,
		400,
		0, 0, 0,
		1,
		0, 0, 5, 0,
		uintptr(unsafe.Pointer(utf16(name))),
	)
	return font
}

func utf16(value string) *uint16 {
	ptr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		ptr, _ = windows.UTF16PtrFromString("")
	}
	return ptr
}
