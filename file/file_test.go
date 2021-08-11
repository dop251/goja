package file

import "testing"

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
