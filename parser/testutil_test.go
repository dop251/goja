package parser

import (
	"fmt"
	"runtime"
	"testing"
	"path/filepath"
)

// Quick and dirty replacement for terst

func tt(t *testing.T, f func()) {
	defer func() {
		if x := recover(); x != nil {
			_, file, line, _ := runtime.Caller(5)
			t.Errorf("Error at %s:%d: %v", filepath.Base(file), line, x)
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


