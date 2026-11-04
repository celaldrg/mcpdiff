package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	var b bytes.Buffer
	code := run([]string{"testdata/old.json", "testdata/new.json"}, &b)
	if code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
	out := b.String()
	for _, want := range []string{"tool removed", "type changed integer -> string", "enum value deep removed", "new required param index", "new optional param lang", "tool added", "description changed", "4 breaking"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestIdentical(t *testing.T) {
	var b bytes.Buffer
	if code := run([]string{"testdata/old.json", "testdata/old.json"}, &b); code != 0 {
		t.Fatalf("want 0, got %d: %s", code, b.String())
	}
}

func TestBareArray(t *testing.T) {
	m, err := parse([]byte(`[{"name":"a"}]`))
	if err != nil || len(m) != 1 {
		t.Fatal(err, m)
	}
}
