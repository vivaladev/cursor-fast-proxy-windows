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
