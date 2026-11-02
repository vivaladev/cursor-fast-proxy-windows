package jsontext

import "testing"

func TestSetPreservesCommentAndReplacesValue(t *testing.T) {
	source := "{\n  // keep\n  \"editor.fontSize\": 14,\n  \"http.proxy\": \"old\"\n}\n"
	got, err := Set(source, "http.proxy", String("http://127.0.0.1:18081"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  // keep\n  \"editor.fontSize\": 14,\n  \"http.proxy\": \"http://127.0.0.1:18081\"\n}\n"; got != want {
		t.Fatalf("got %q", got)
	}
}

func TestSetInsertsBeforeClosingBrace(t *testing.T) {
	got, err := Set("{\n  \"editor.fontSize\": 14\n}\n", "http.proxySupport", String("override"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"editor.fontSize\": 14\n,\n  \"http.proxySupport\": \"override\"\n}\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestSetEmptyObject(t *testing.T) {
	got, err := Set("{\n}\n", "cursor.general.disableHttp2", Bool(true))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n\n  \"cursor.general.disableHttp2\": true\n}\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestSetBoolReplacesString(t *testing.T) {
	got, err := Set(`{"cursor.general.disableHttp2": false}`, "cursor.general.disableHttp2", Bool(true))
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"cursor.general.disableHttp2": true}` {
		t.Fatalf("got %q", got)
	}
}

func TestRemoveKeepsOtherKeysAndComments(t *testing.T) {
	source := "{\n  // keep\n  \"editor.fontSize\": 14,\n  \"http.proxy\": \"http://127.0.0.1:18081\",\n  \"workbench.colorTheme\": \"dark\"\n}\n"
	got, err := Remove(source, "http.proxy")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  // keep\n  \"editor.fontSize\": 14,\n  \n  \"workbench.colorTheme\": \"dark\"\n}\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestRemoveLastPropertyDropsComma(t *testing.T) {
	got, err := Remove("{\n  \"a\": 1,\n  \"http.proxy\": \"x\"\n}\n", "http.proxy")
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("got %q", got)
	}
}

func TestRemoveMissingKey(t *testing.T) {
	source := "{ \"a\": 1 }\n"
	got, err := Remove(source, "http.proxy")
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("got %q", got)
	}
}

func TestRejectsArrayAndBrokenObject(t *testing.T) {
	if _, err := Set("[]", "a", String("b")); err == nil || err.Error() != "JSON не объект" {
		t.Fatalf("array err %v", err)
	}
	if _, err := Set(`{"a" 1}`, "a", String("b")); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

func TestBlockCommentSurvives(t *testing.T) {
	source := "{ /* note */ \"a\": 1 }\n"
	got, err := Set(source, "b", String("c"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{ /* note */ \"a\": 1 ,\n  \"b\": \"c\"\n}\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestEscapesInsertedString(t *testing.T) {
	got, err := Set("{}", "k", String("a\"b\\c"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\n  \"k\": \"a\\\"b\\\\c\"\n}" {
		t.Fatalf("got %q", got)
	}
}
