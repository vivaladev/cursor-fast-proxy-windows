package main

import "testing"

func TestControlIDsUnique(t *testing.T) {
	seen := map[int]string{}
	for _, c := range controls() {
		if prev, ok := seen[c.ID]; ok {
			t.Fatalf("id %d у %s и %s", c.ID, prev, c.Name)
		}
		seen[c.ID] = c.Name
		if c.CreateVisible {
			t.Fatalf("%s создаётся видимым и может перехватить клик", c.Name)
		}
	}
}

func TestSettingsClickLandsInInputs(t *testing.T) {
	for _, id := range []int{idIP, idUser, idPass} {
		field, ok := controlByID(id)
		if !ok || !field.Input || field.ReadOnly || !field.on("settings") {
			t.Fatalf("поле %d не настроено как ввод", id)
		}
		if !inputFitsCaret(field) {
			t.Fatalf("%s высотой %d не вмещает курсор", field.Name, field.H)
		}
		points := [][2]int{
			{field.X + field.W/2, field.Y + field.H/2},
			{field.X + 3, field.Y + 3},
			{field.X + field.W - 4, field.Y + field.H - 4},
		}
		for _, p := range points {
			got, hit := hitControl("settings", p[0], p[1])
			if !hit || got.ID != id {
				t.Fatalf("клик %d,%d по %s попал в %s", p[0], p[1], field.Name, got.Name)
			}
		}
	}
	pass, _ := controlByID(idPass)
	if !pass.Password {
		t.Fatal("password без маски")
	}
}

func TestSharedRectanglesFollowTheOpenScreen(t *testing.T) {
	pass, _ := controlByID(idPass)
	button, _ := controlByID(idSettings)
	logBox, _ := controlByID(idLogBox)
	x1 := max(pass.X, button.X)
	y1 := max(pass.Y, button.Y)
	x2 := min(pass.X+pass.W, button.X+button.W)
	y2 := min(pass.Y+pass.H, button.Y+button.H)
	if x1 >= x2 || y1 >= y2 {
		t.Fatal("кнопка «Настройки» больше не пересекает поле password")
	}
	x, y := (x1+x2)/2, (y1+y2)/2
	if got, ok := hitControl("settings", x, y); !ok || got.ID != idPass {
		t.Fatalf("на настройках клик попал в %s", got.Name)
	}
	if got, ok := hitControl("main", x, y); !ok || got.ID != idSettings {
		t.Fatalf("на главном экране клик попал в %s", got.Name)
	}
	for _, id := range []int{idIP, idUser, idPass} {
		field, _ := controlByID(id)
		cx, cy := field.X+field.W/2, field.Y+field.H/2
		if !logBox.contains(cx, cy) {
			t.Fatalf("журнал не перекрывает %s", field.Name)
		}
		if got, ok := hitControl("settings", cx, cy); !ok || got.ID != id {
			t.Fatalf("журнал перехватил клик по %s (%s)", field.Name, got.Name)
		}
		if got, ok := hitControl("logs", cx, cy); !ok || got.ID != idLogBox {
			t.Fatalf("на экране логов клик попал в %s", got.Name)
		}
	}
}
