package file

import (
	"testing"
)

func TestPosition(t *testing.T) {
	const SRC = `line1
line2
line3`
	f := NewFile("", SRC, 0)

	tests := []struct {
		offset int
		line   int
		col    int
	}{
		{0, 1, 1},
		{2, 1, 3},
		{2, 1, 3},
		{6, 2, 1},
		{7, 2, 2},
		{12, 3, 1},
		{12, 3, 1},
		{13, 3, 2},
		{13, 3, 2},
		{16, 3, 5},
		{17, 3, 6},
	}

	for i, test := range tests {
		if p := f.Position(test.offset); p.Line != test.line || p.Column != test.col {
			t.Fatalf("%d. Line: %d, col: %d", i, p.Line, p.Column)
		}
	}
}

func TestFileConcurrency(t *testing.T) {
	const SRC = `line1
line2
line3`
	f := NewFile("", SRC, 0)
	go func() {
		f.Position(12)
	}()
	f.Position(2)
}

func TestGetSourceFilename(t *testing.T) {
	tests := []struct {
		source, basename, result string
	}{
		{"test.js", "base.js", "test.js"},
		{"test.js", "../base.js", "../test.js"},
		{"test.js", "/somewhere/base.js", "/somewhere/test.js"},
		{"/test.js", "/somewhere/base.js", "/test.js"},
		{"/test.js", "file:///somewhere/base.js", "file:///test.js"},
		{"file:///test.js", "base.js", "file:///test.js"},
		{"file:///test.js", "/somwehere/base.js", "file:///test.js"},
		{"file:///test.js", "file:///somewhere/base.js", "file:///test.js"},
		{"../test.js", "/somewhere/else/base.js", "/somewhere/test.js"},
		{"../test.js", "file:///somewhere/else/base.js", "file:///somewhere/test.js"},
		{"../test.js", "https://example.com/somewhere/else/base.js", "https://example.com/somewhere/test.js"},
		{"\ntest.js", "base123.js", "test.js"},
		{"\rtest2.js\t\n  ", "base123.js", "test2.js"},
		// TODO find something that won't parse
	}
	for _, test := range tests {
		resultURL := ResolveSourcemapURL(test.basename, test.source)
		result := resultURL.String()
		if result != test.result {
			t.Fatalf("source: %q, basename %q produced %q instead of %q", test.source, test.basename, result, test.result)
		}
	}
}
