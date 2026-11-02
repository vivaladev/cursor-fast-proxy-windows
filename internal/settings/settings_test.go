package settings

import "testing"

func TestRoundTripQuotesPassword(t *testing.T) {
	in := Credentials{IP: "203.0.113.10", User: "root", Password: "a'b c"}
	text, err := Write(in)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := Read(text)
	if err != nil || !ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
	if got != in {
		t.Fatalf("got %+v", got)
	}
}

func TestRejectsSpacesInIPAndEmptyPassword(t *testing.T) {
	if _, err := Write(Credentials{IP: "203.0.113.10 bad", User: "root", Password: "x"}); err == nil {
		t.Fatal("ip со пробелом")
	}
	if _, err := Write(Credentials{IP: "203.0.113.10", User: "root", Password: ""}); err == nil {
		t.Fatal("пустой пароль")
	}
	if _, err := Write(Credentials{IP: "203.0.113.10", User: "ro ot", Password: "x"}); err == nil {
		t.Fatal("user со пробелом")
	}
}

func TestPasswordMayContainSpaces(t *testing.T) {
	_, err := Write(Credentials{IP: "203.0.113.8", User: "alice", Password: "two words"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMissingBlock(t *testing.T) {
	_, ok, err := Read("export PATH=\"$PATH\"\n")
	if err != nil || ok {
		t.Fatalf("ok %v err %v", ok, err)
	}
}

func TestBrokenMarker(t *testing.T) {
	_, _, err := Read("# >>> cursor-fast-proxy >>>\nexport PATH\n")
	if err == nil {
		t.Fatal("ожидался оборванный блок")
	}
}

func TestRejectsNewlineInField(t *testing.T) {
	if err := Validate(Credentials{IP: "a\nb", User: "u", Password: "p"}); err == nil {
		t.Fatal("перевод строки в ip")
	}
}
