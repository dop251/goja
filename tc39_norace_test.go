//go:build !race
// +build !race

package goja

import "testing"

// Prevent linter warnings about unused type
var _ = tc39Test{name: "", f: nil}

func (ctx *tc39TestCtx) runTest(name string, f func(t *testing.T)) {
	ctx.t.Run(name, func(t *testing.T) {
		t.Parallel()
		f(t)
	})
}

func (ctx *tc39TestCtx) flush() {
}
