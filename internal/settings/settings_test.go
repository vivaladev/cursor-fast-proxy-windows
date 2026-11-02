package settings

import "testing"

func TestRoundTripPasswordWithSpaces(t *testing.T) {
	text, err := Write(Credentials{IP: "203.0.113.10", User: "root", Password: "two words"})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := Read(text)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if got.Password != "two words" || got.User != "root" || got.IP != "203.0.113.10" {
		t.Fatalf("%+v", got)
	}
}

func TestQuoteContainsApostrophe(t *testing.T) {
	text, err := Write(Credentials{IP: "2001:db8::1", User: "deploy", Password: "o'clock"})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := Read(text)
	if err != nil || !ok || got.Password != "o'clock" {
		t.Fatalf("ok=%v err=%v pass=%q", ok, err, got.Password)
	}
}

func TestRejectsBadIP(t *testing.T) {
	if err := Validate(Credentials{IP: "not an ip", User: "root", Password: "x"}); err == nil {
		t.Fatal("ждали ошибку ip")
	}
}

func TestEmptyFile(t *testing.T) {
	_, ok, err := Read("")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
