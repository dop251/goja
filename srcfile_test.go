package goja

import "testing"

func TestPosition(t *testing.T) {
	const SRC = `line1
line2
line3`
	f := NewSrcFile("", SRC)
	if p := f.Position(12); p.Line != 3 || p.Col != 1 {
		t.Fatalf("0. Line: %d, col: %d", p.Line, p.Col)
	}

	if p := f.Position(2); p.Line != 1 || p.Col != 3 {
		t.Fatalf("1. Line: %d, col: %d", p.Line, p.Col)
	}

	if p := f.Position(2); p.Line != 1 || p.Col != 3 {
		t.Fatalf("2. Line: %d, col: %d", p.Line, p.Col)
	}

	if p := f.Position(7); p.Line != 2 || p.Col != 2 {
		t.Fatalf("3. Line: %d, col: %d", p.Line, p.Col)
	}

	if p := f.Position(12); p.Line != 3 || p.Col != 1 {
		t.Fatalf("4. Line: %d, col: %d", p.Line, p.Col)
	}

}
