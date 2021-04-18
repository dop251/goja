package parser

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

// Quick and dirty replacement for terst

func tt(t *testing.T, f func()) {
	defer func() {
		if x := recover(); x != nil {
			pcs := make([]uintptr, 16)
			pcs = pcs[:runtime.Callers(1, pcs)]
			frames := runtime.CallersFrames(pcs)
			var file string
			var line int
			for {
				frame, more := frames.Next()
				// The line number here must match the line where f() is called (see below)
				if frame.Line == 40 && filepath.Base(frame.File) == "testutil_test.go" {
					break
				}

				if !more {
					break
				}
				file, line = frame.File, frame.Line
			}
			if line > 0 {
				t.Errorf("Error at %s:%d: %v", filepath.Base(file), line, x)
			} else {
				t.Errorf("Error at <unknown>: %v", x)
			}
		}
	}()

	f()
}

func is(a, b interface{}) {
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	if as != bs {
		panic(fmt.Errorf("%+v(%T) != %+v(%T)", a, a, b, b))
	}
}
