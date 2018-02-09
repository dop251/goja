package goja

import (
	"fmt"
	"sort"
	"strings"
)

type Position struct {
	Line, Col int
}

type SrcFile struct {
	name string
	src  string

	lineOffsets       []int
	lastScannedOffset int
}

func NewSrcFile(name, src string) *SrcFile {
	return &SrcFile{
		name: name,
		src:  src,
	}
}

func (f *SrcFile) Position(offset int) Position {
	var line int
	if offset > f.lastScannedOffset {
		line = f.scanTo(offset)
	} else {
		line = sort.Search(len(f.lineOffsets), func(x int) bool { return f.lineOffsets[x] > offset }) - 1
	}

	var lineStart int
	if line >= 0 {
		lineStart = f.lineOffsets[line]
	}
	return Position{
		Line: line + 2,
		Col:  offset - lineStart + 1,
	}
}

func (f *SrcFile) scanTo(offset int) int {
	o := f.lastScannedOffset
	for o < offset {
		p := strings.Index(f.src[o:], "\n")
		if p == -1 {
			f.lastScannedOffset = len(f.src)
			return len(f.lineOffsets) - 1
		}
		o = o + p + 1
		f.lineOffsets = append(f.lineOffsets, o)
	}
	f.lastScannedOffset = o

	if o == offset {
		return len(f.lineOffsets) - 1
	}

	return len(f.lineOffsets) - 2
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}
