//go:build windows

package main

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/vivaladev/cursor-fast-proxy-windows/internal/apppaths"
	"github.com/vivaladev/cursor-fast-proxy-windows/internal/workflow"
)

func TestSettingsFieldsAcceptCaret(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	dir := t.TempDir()
	desk, err := openDesk(&workflow.Engine{Paths: apppaths.Paths{Home: dir, Roaming: dir, Local: dir}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		user32.NewProc("DestroyWindow").Call(desk.hwnd)
		state = nil
	})
	pump(40)

	button, _ := controlByID(idSettings)
	if got := childAt(desk.hwnd, button.X+button.W/2, button.Y+button.H/2); got != desk.byID[idSettings] {
		t.Fatalf("на главном экране клик по «Настройки» попал в %s", controlName(got))
	}
	if pass, ok := controlByID(idPass); !ok || !button.contains(pass.X+pass.W/2, pass.Y+pass.H/2) {
		t.Fatal("кнопка «Настройки» больше не перекрывает поле password")
	}
	if visible(desk.byID[idPass]) {
		t.Fatal("поле password видно на главном экране")
	}

	desk.show("settings")
	pump(40)
	if visible(desk.logBox) {
		t.Fatal("журнал остался поверх полей")
	}
	for _, id := range []int{idIP, idUser, idPass} {
		field, _ := controlByID(id)
		hwnd := desk.byID[id]
		if hwnd == 0 {
			t.Fatalf("%s не создано", field.Name)
		}
		if !visible(hwnd) {
			t.Fatalf("%s скрыто", field.Name)
		}
		x, y, w, h := windowInParent(desk.hwnd, hwnd)
		if x != field.X || y != field.Y || w != field.W || h != field.H {
			t.Fatalf("%s на экране %d,%d %dx%d, в раскладке %d,%d %dx%d", field.Name, x, y, w, h, field.X, field.Y, field.W, field.H)
		}
		if got := childAt(desk.hwnd, field.X+field.W/2, field.Y+field.H/2); got != hwnd {
			t.Fatalf("клик по %s попал в %s", field.Name, controlName(got))
		}
		click(hwnd, field.W/2, field.H/2)
		if focus := focused(); focus != hwnd {
			t.Fatalf("после клика фокус у %s, ждали %s", controlName(focus), field.Name)
		}
		if !caretVisible() {
			t.Fatalf("в %s нет курсора", field.Name)
		}
		pos, _, _ := procSendMessageW.Call(hwnd, emCharFromPos, 0, packPoint(8, field.H/2))
		if uint16(pos) == 0xFFFF {
			t.Fatalf("%s не принимает позицию курсора", field.Name)
		}
		procSendMessageW.Call(hwnd, wmChar, uintptr('Z'), 0)
		if getText(hwnd) != "Z" {
			t.Fatalf("%s не приняло ввод, текст %q", field.Name, getText(hwnd))
		}
	}

	label, _ := controlByID(idUserLabel)
	click(desk.byID[idUserLabel], label.W/2, label.H/2)
	if focus := focused(); focus != desk.byID[idUser] {
		t.Fatalf("клик по подписи user поставил фокус в %s", controlName(focus))
	}
}

func childAt(parent uintptr, x, y int) uintptr {
	hwnd, _, _ := user32.NewProc("RealChildWindowFromPoint").Call(parent, packPoint(x, y))
	return hwnd
}

func focused() uintptr {
	hwnd, _, _ := procGetFocus.Call()
	return hwnd
}

func visible(hwnd uintptr) bool {
	r, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd)
	return r != 0
}

func caretVisible() bool {
	var pt point
	ok, _, _ := user32.NewProc("GetCaretPos").Call(uintptr(unsafe.Pointer(&pt)))
	runtime.KeepAlive(&pt)
	return ok != 0
}

func click(hwnd uintptr, x, y int) {
	pt := packPoint(x, y)
	procPostMessageW.Call(hwnd, wmLButtonDown, 1, pt)
	procPostMessageW.Call(hwnd, wmLButtonUp, 0, pt)
	pump(40)
}

func windowInParent(parent, hwnd uintptr) (x, y, w, h int) {
	var r struct{ Left, Top, Right, Bottom int32 }
	user32.NewProc("GetWindowRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	user32.NewProc("MapWindowPoints").Call(0, parent, uintptr(unsafe.Pointer(&r)), 2)
	runtime.KeepAlive(&r)
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func packPoint(x, y int) uintptr {
	return uintptr(uint32(int32(x))) | uintptr(uint32(int32(y)))<<32
}

func controlName(hwnd uintptr) string {
	if hwnd == 0 {
		return "пусто"
	}
	id, _, _ := user32.NewProc("GetDlgCtrlID").Call(hwnd)
	if c, ok := controlByID(int(id)); ok {
		return c.Name
	}
	if state != nil && hwnd == state.hwnd {
		return "окно"
	}
	return "чужое окно"
}
