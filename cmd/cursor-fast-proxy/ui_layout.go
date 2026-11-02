package main

// uiFontPx is the Segoe UI height passed to CreateFont (character height).
const uiFontPx = 18

const (
	idApply     = 101
	idRemove    = 102
	idSettings  = 103
	idLogs      = 104
	idSave      = 105
	idBack      = 106
	idYes       = 107
	idYesVPS    = 108
	idIP        = 201
	idUser      = 202
	idPass      = 203
	idIPLabel   = 211
	idUserLabel = 212
	idPassLabel = 213
	idStatus    = 301
	idLogBox    = 302
	idHint      = 303
)

// control is one child window. Screens share one parent, so rectangles of
// different screens overlap. stack puts the active screen above the rest,
// which is what lets a click land in ip/user/password.
type control struct {
	Name          string
	ID            int
	Kind          string
	Text          string
	X, Y, W, H    int
	Screen        string
	Input         bool
	Password      bool
	ReadOnly      bool
	Multi         bool
	TabStop       bool
	CreateVisible bool
}

func (c control) on(screen string) bool {
	switch c.Screen {
	case "chrome":
		return true
	case "back":
		return screen != "main"
	default:
		return c.Screen == screen
	}
}

func (c control) contains(x, y int) bool {
	return x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H
}

func controls() []control {
	return []control{
		{Name: "hint", ID: idHint, Kind: "hint", Text: "Cursor ходит в облако через 127.0.0.1:18081. Остальной трафик Windows не трогается.", X: 16, Y: 16, W: 440, H: 48, Screen: "chrome"},
		{Name: "status", ID: idStatus, Kind: "edit", X: 16, Y: 72, W: 440, H: 120, Screen: "chrome", ReadOnly: true, Multi: true},

		{Name: "apply", ID: idApply, Kind: "button", Text: "Вжух и прокси", X: 16, Y: 210, W: 440, H: 40, Screen: "main", TabStop: true},
		{Name: "remove", ID: idRemove, Kind: "button", Text: "Удалить настройки прокси", X: 16, Y: 258, W: 440, H: 40, Screen: "main", TabStop: true},
		{Name: "settings", ID: idSettings, Kind: "button", Text: "Настройки", X: 16, Y: 306, W: 440, H: 40, Screen: "main", TabStop: true},
		{Name: "logs", ID: idLogs, Kind: "button", Text: "Логи", X: 16, Y: 354, W: 440, H: 40, Screen: "main", TabStop: true},

		{Name: "ip-label", ID: idIPLabel, Kind: "label", Text: "ip", X: 16, Y: 216, W: 90, H: 24, Screen: "settings"},
		{Name: "ip", ID: idIP, Kind: "edit", X: 112, Y: 208, W: 344, H: 40, Screen: "settings", Input: true, TabStop: true},
		{Name: "user-label", ID: idUserLabel, Kind: "label", Text: "user", X: 16, Y: 264, W: 90, H: 24, Screen: "settings"},
		{Name: "user", ID: idUser, Kind: "edit", X: 112, Y: 256, W: 344, H: 40, Screen: "settings", Input: true, TabStop: true},
		{Name: "password-label", ID: idPassLabel, Kind: "label", Text: "password", X: 16, Y: 312, W: 90, H: 24, Screen: "settings"},
		{Name: "password", ID: idPass, Kind: "edit", X: 112, Y: 304, W: 344, H: 40, Screen: "settings", Input: true, Password: true, TabStop: true},
		{Name: "save", ID: idSave, Kind: "button", Text: "Сохранить", X: 16, Y: 360, W: 440, H: 40, Screen: "settings", TabStop: true},

		{Name: "yes", ID: idYes, Kind: "button", Text: "Да", X: 16, Y: 210, W: 440, H: 40, Screen: "confirm", TabStop: true},
		{Name: "yes-vps", ID: idYesVPS, Kind: "button", Text: "Да, и на vps", X: 16, Y: 258, W: 440, H: 40, Screen: "confirm", TabStop: true},

		{Name: "log", ID: idLogBox, Kind: "edit", X: 16, Y: 208, W: 440, H: 250, Screen: "logs", ReadOnly: true, Multi: true},
		{Name: "back", ID: idBack, Kind: "button", Text: "Назад", X: 16, Y: 480, W: 440, H: 40, Screen: "back", TabStop: true},
	}
}

func controlByID(id int) (control, bool) {
	for _, c := range controls() {
		if c.ID == id {
			return c, true
		}
	}
	return control{}, false
}

// stack is bottom-to-top. Hidden screens stay under the active one, so a
// click in a shared rectangle hits the field the user sees.
func stack(screen string) []control {
	all := controls()
	out := make([]control, 0, len(all))
	shown := make([]control, 0, len(all))
	for _, c := range all {
		if c.on(screen) {
			shown = append(shown, c)
		} else {
			out = append(out, c)
		}
	}
	return append(out, shown...)
}

func hitControl(screen string, x, y int) (control, bool) {
	var found control
	ok := false
	for _, c := range stack(screen) {
		if c.contains(x, y) {
			found = c
			ok = true
		}
	}
	return found, ok
}

func inputFitsCaret(c control) bool {
	if !c.Input || c.ReadOnly {
		return false
	}
	// Sunken edge eats 2px on each side. The rest has to hold the font and a caret.
	return c.H-4 >= uiFontPx+8 && c.W >= 120
}
