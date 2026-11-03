//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/workflow"
	"golang.org/x/sys/windows"
)

const (
	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsMinimizeBox   = 0x00020000
	wsClipChildren  = 0x02000000
	wsClipSiblings  = 0x04000000
	wsVisible       = 0x10000000
	wsChild         = 0x40000000
	wsTabStop       = 0x00010000
	wsVScroll       = 0x00200000
	wsExClientEdge  = 0x00000200
	ssNotify        = 0x0100
	esAutohscroll   = 0x0080
	esPassword      = 0x0020
	esMultiline     = 0x0004
	esReadonly      = 0x0800
	esAutovscroll   = 0x0040
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmSetFont       = 0x0030
	wmSetFocus      = 0x0007
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmLButtonUp     = 0x0202
	wmLButtonDblclk = 0x0203
	wmChar          = 0x0102
	wmCommand       = 0x0111
	emGetSel        = 0x00B0
	emSetSel        = 0x00B1
	emPosFromChar   = 0x00D6
	emCharFromPos   = 0x00D7
	emSetPassword   = 0x00CC
	emSetLimitText  = 0x00C5
	swShow          = 5
	swpNoActivate   = 0x0010
	swpShowWindow   = 0x0040
	swpHideWindow   = 0x0080
	wmStatus        = 0x8000 + 1
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procPeekMessageW       = user32.NewProc("PeekMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procIsDialogMessageW   = user32.NewProc("IsDialogMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procShowWindow         = user32.NewProc("ShowWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW  = user32.NewProc("GetWindowTextLengthW")
	procLoadCursorW        = user32.NewProc("LoadCursorW")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procEnableWindow       = user32.NewProc("EnableWindow")
	procSetFocus           = user32.NewProc("SetFocus")
	procGetFocus           = user32.NewProc("GetFocus")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procShowCaret          = user32.NewProc("ShowCaret")
	procSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW    = user32.NewProc("CallWindowProcW")
	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procCreateFontW        = gdi32.NewProc("CreateFontW")

	windowCallback = windows.NewCallback(windowProc)
	inputCallback  = windows.NewCallback(inputProc)
	origInputProc  uintptr
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
	engine      *workflow.Engine
	hwnd        uintptr
	font        uintptr
	status      uintptr
	hint        uintptr
	logBox      uintptr
	byID        map[int]uintptr
	interactive []uintptr
	screen      string
}

func runWindow(engine *workflow.Engine) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if _, err := openDesk(engine); err != nil {
		return err
	}
	var msg message
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		dispatch(&msg)
	}
	return nil
}

func openDesk(engine *workflow.Engine) (*desk, error) {
	enableDPI()
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
		runtime.KeepAlive(className)
		return nil, fmt.Errorf("окно: %v", err)
	}
	title := utf16(windowTitle())
	state = &desk{engine: engine, screen: "main", byID: map[int]uintptr{}}
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsCaption|wsSysMenu|wsMinimizeBox|wsClipChildren,
		120, 80, 480, 680,
		0, 0, instance, 0,
	)
	runtime.KeepAlive(className)
	runtime.KeepAlive(title)
	if hwnd == 0 {
		return nil, fmt.Errorf("окно не создалось: %v", err)
	}
	state.hwnd = hwnd
	state.font = makeFont("Segoe UI", -uiFontPx)
	state.build(instance)
	// The first ShowWindow in the process can be replaced by STARTUPINFO.
	// The second call is the one that actually shows the window.
	procShowWindow.Call(hwnd, swShow)
	procShowWindow.Call(hwnd, 1)
	state.show("main")
	setText(state.status, "Впиши ip, user и password, сохрани и нажми «Вжух и прокси».")
	return state, nil
}

func enableDPI() {
	proc := user32.NewProc("SetProcessDpiAwarenessContext")
	if err := proc.Find(); err == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
		if r, _, _ := proc.Call(^uintptr(3)); r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
}

func (d *desk) build(instance uintptr) {
	for _, c := range controls() {
		hwnd := d.child(c, instance)
		d.byID[c.ID] = hwnd
		switch c.ID {
		case idHint:
			d.hint = hwnd
		case idStatus:
			d.status = hwnd
			limit(hwnd)
		case idLogBox:
			d.logBox = hwnd
			limit(hwnd)
		}
		if c.Kind == "button" || c.Input {
			d.interactive = append(d.interactive, hwnd)
		}
		if c.Input {
			subclassInput(hwnd)
			if c.Password {
				procSendMessageW.Call(hwnd, emSetPassword, uintptr('*'), 0)
			}
		}
	}
}

func (d *desk) child(c control, instance uintptr) uintptr {
	class := "STATIC"
	extra := uintptr(0)
	ex := uintptr(0)
	switch c.Kind {
	case "button":
		class = "BUTTON"
	case "edit":
		class = "EDIT"
		extra |= esAutohscroll
		ex = wsExClientEdge
		if c.Password {
			extra |= esPassword
		}
		if c.Multi {
			extra |= wsVScroll | esMultiline | esAutovscroll
		}
		if c.ReadOnly {
			extra |= esReadonly
		}
	case "label":
		extra |= ssNotify
	}
	style := uintptr(wsChild | wsClipSiblings)
	if c.TabStop {
		style |= wsTabStop
	}
	if c.CreateVisible {
		style |= wsVisible
	}
	style |= extra
	cls := utf16(class)
	txt := utf16(c.Text)
	hwnd, _, _ := procCreateWindowExW.Call(
		ex,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(txt)),
		style,
		uintptr(c.X), uintptr(c.Y), uintptr(c.W), uintptr(c.H),
		d.hwnd,
		uintptr(c.ID),
		instance,
		0,
	)
	runtime.KeepAlive(cls)
	runtime.KeepAlive(txt)
	if d.font != 0 && hwnd != 0 {
		procSendMessageW.Call(hwnd, wmSetFont, d.font, 1)
	}
	return hwnd
}

func (d *desk) show(screen string) {
	d.screen = screen
	for _, c := range stack(screen) {
		hwnd := d.byID[c.ID]
		on := c.on(screen)
		flag := uintptr(0)
		if on {
			flag = 1
		}
		procEnableWindow.Call(hwnd, flag)
		place(hwnd, c.placeOn(screen), on)
		if on && c.Input && d.font != 0 {
			// The font was applied while the field was hidden. Windows 10 then
			// keeps a text line the click cannot hit until the font is set again.
			procSendMessageW.Call(hwnd, wmSetFont, d.font, 1)
		}
	}
	switch screen {
	case "main":
		setText(d.hint, "Cursor ходит в облако через 127.0.0.1:18081. Остальной трафик Windows не трогается.")
	case "settings":
		creds, err := loadSettings(d.engine)
		if err != nil {
			setText(d.status, err.Error())
		}
		setByID(idIP, creds.IP)
		setByID(idUser, creds.User)
		setByID(idPass, creds.Password)
		setText(d.hint, "Сохранить пишет блок в ~\\.cursor-fast-proxy\\settings.env и не ставит системный прокси.")
		focusAtEnd(d.byID[idIP])
	case "confirm":
		setText(d.hint, "«Да» убирает локальный прокси. «Да, и на vps» ещё удаляет сервис sing-box-cursor.")
	case "logs":
		setText(d.logBox, readLog(d.engine))
		setText(d.hint, "В журнале ip, user и password заменены именами переменных.")
	}
}

func place(hwnd uintptr, c control, visible bool) {
	if hwnd == 0 {
		return
	}
	insert := uintptr(1) // HWND_BOTTOM
	flags := uintptr(swpNoActivate)
	if visible {
		insert = 0 // HWND_TOP
		flags |= swpShowWindow
	} else {
		flags |= swpHideWindow
	}
	procSetWindowPos.Call(
		hwnd,
		insert,
		uintptr(uint32(int32(c.X))),
		uintptr(uint32(int32(c.Y))),
		uintptr(c.W),
		uintptr(c.H),
		flags,
	)
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
	case idIPLabel:
		focusAtEnd(d.byID[idIP])
	case idUserLabel:
		focusAtEnd(d.byID[idUser])
	case idPassLabel:
		focusAtEnd(d.byID[idPass])
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
	for _, hwnd := range d.interactive {
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

func inputProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmLButtonDown, wmLButtonUp, wmLButtonDblclk:
		lParam = aimAtText(hwnd, lParam)
	case wmMouseMove:
		if wParam&1 != 0 {
			lParam = aimAtText(hwnd, lParam)
		}
	}
	ret := callInput(hwnd, msg, wParam, lParam)
	if msg == wmSetFocus {
		procShowCaret.Call(hwnd)
	}
	return ret
}

func callInput(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if origInputProc == 0 {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
	ret, _, _ := procCallWindowProcW.Call(origInputProc, hwnd, uintptr(msg), wParam, lParam)
	return ret
}

type clientRect struct{ Left, Top, Right, Bottom int32 }

func aimAtText(hwnd uintptr, lParam uintptr) uintptr {
	x, y := mousePoint(lParam)
	lineY, lineH := textLine(hwnd)
	var client clientRect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	runtime.KeepAlive(&client)
	nx, ny := clickOnTextLine(int(x), int(y), lineY, lineH, int(client.Bottom-client.Top))
	return packMouse(int32(nx), int32(ny))
}

func textLine(hwnd uintptr) (y, h int) {
	pos, _, _ := procSendMessageW.Call(hwnd, emPosFromChar, 0, 0)
	y = int(int32(pos >> 32))
	h = uiFontPx
	return y, h
}

func mousePoint(lParam uintptr) (x, y int32) {
	v := uint32(lParam)
	return int32(int16(v & 0xFFFF)), int32(int16(v >> 16))
}

func packMouse(x, y int32) uintptr {
	return uintptr(uint16(x)) | uintptr(uint16(y))<<16
}

func subclassInput(hwnd uintptr) {
	prev, _, _ := procSetWindowLongPtrW.Call(hwnd, ^uintptr(3), inputCallback)
	if prev != 0 && prev != inputCallback && origInputProc == 0 {
		origInputProc = prev
	}
}

func focusAtEnd(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	procSetFocus.Call(hwnd)
	n, _, _ := procGetWindowTextLenW.Call(hwnd)
	procSendMessageW.Call(hwnd, emSetSel, n, n)
}

func dispatch(msg *message) {
	if state != nil && dialogCanTake(msg.Message) {
		handled, _, _ := procIsDialogMessageW.Call(state.hwnd, uintptr(unsafe.Pointer(msg)))
		if handled != 0 {
			return
		}
	}
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func pump(limit int) {
	var msg message
	for i := 0; i < limit; i++ {
		r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if r == 0 {
			return
		}
		dispatch(&msg)
	}
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
	ptr := utf16(text)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(ptr)))
	runtime.KeepAlive(ptr)
}

func setByID(id int, text string) {
	setText(find(id), text)
}

func textByID(id int) string {
	return getText(find(id))
}

func find(id int) uintptr {
	if state == nil {
		return 0
	}
	return state.byID[id]
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

func limit(hwnd uintptr) {
	procSendMessageW.Call(hwnd, emSetLimitText, 1<<20, 0)
}

func makeFont(name string, height int32) uintptr {
	namePtr := utf16(name)
	font, _, _ := procCreateFontW.Call(
		uintptr(uint32(height)),
		0, 0, 0,
		400,
		0, 0, 0,
		1,
		0, 0, 5, 0,
		uintptr(unsafe.Pointer(namePtr)),
	)
	runtime.KeepAlive(namePtr)
	return font
}

func utf16(value string) *uint16 {
	ptr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		ptr, _ = windows.UTF16PtrFromString("")
	}
	return ptr
}
