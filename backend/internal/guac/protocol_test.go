package guac

import (
	"bufio"
	"strings"
	"testing"
)

func TestEncodeCountsCodePoints(t *testing.T) {
	got := Encode("name", "سلام", "")
	if want := "4.name,4.سلام,0.;"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReaderRoundTrip(t *testing.T) {
	in := Encode("args", "VERSION_1_5_0", "hostname") + Encode("ready", "$abc") + Encode("", "ü;,.")
	rd := NewReader(bufio.NewReader(strings.NewReader(in)))
	for _, want := range [][]string{{"args", "VERSION_1_5_0", "hostname"}, {"ready", "$abc"}, {"", "ü;,."}} {
		raw, elems, err := rd.Read()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(elems, "|") != strings.Join(want, "|") || raw != Encode(want...) {
			t.Fatalf("got %q / %v", raw, elems)
		}
	}
}

func TestReaderRejectsGarbage(t *testing.T) {
	for _, in := range []string{"x.abc;", ".a;", "3.abc!", "99999999999.a;"} {
		rd := NewReader(bufio.NewReader(strings.NewReader(in)))
		if _, _, err := rd.Read(); err == nil {
			t.Fatalf("%q: expected error", in)
		}
	}
}

func TestUploadNames(t *testing.T) {
	tests := []struct {
		name  string
		frame string
		want  string
	}{
		{"single upload", Encode("file", "1", "text/plain", "report.txt"), "report.txt"},
		{"batched with mouse", Encode("mouse", "1", "2", "0") + Encode("file", "2", "application/pdf", "ب.pdf"), "ب.pdf"},
		{"no file", Encode("key", "65", "1"), ""},
		{"garbage", "4.file,1.", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.Join(uploadNames(tc.frame), ","); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
