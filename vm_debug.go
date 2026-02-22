package goja

import (
	"sync/atomic"

	"github.com/dop251/goja/unistring"
)

// dbgScopeInfo tracks an active scope's stack-register variables for the debugger.
// Each variable maps to an absolute index in vm.stack, computed at scope entry time.
type dbgScopeInfo struct {
	vars map[unistring.String]int // variable name → absolute stack index
}

// debuggerInstr implements the JS `debugger` statement.
// When a Debugger is attached it invokes the debug hook; otherwise it is a no-op.
type debuggerInstr struct{}

func (debuggerInstr) exec(vm *vm) {
	if vm.dbg != nil && vm.dbg.hook != nil {
		vm.dbg.invokeHook(vm, DebugEventDebuggerStmt)
	}
	vm.pc++
}

// currentPosition resolves the VM's current program counter to a source position.
func (vm *vm) currentPosition() DebugPosition {
	if vm.prg == nil || vm.prg.src == nil {
		return DebugPosition{}
	}
	pos := vm.prg.src.Position(vm.prg.sourceOffset(vm.pc))
	return DebugPosition{
		Filename: pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
	}
}

// runWithDebugger is the debug-aware execution loop, dispatched from run()
// when a Debugger is attached. It checks for breakpoints and stepping at
// statement boundaries.
func (vm *vm) runWithDebugger() {
	dbg := vm.dbg
	interrupted := false
	for {
		if interrupted = atomic.LoadUint32(&vm.interrupted) != 0; interrupted {
			break
		}

		// Check for user-initiated pause request
		if atomic.CompareAndSwapUint32(&dbg.pauseRequested, 1, 0) {
			dbg.invokeHook(vm, DebugEventPause)
		}

		pc := vm.pc
		if pc < 0 || pc >= len(vm.prg.code) {
			break
		}

		// Execute function/block entry instructions before boundary detection
		// so that debug scopes are initialized before the debugger can pause.
		// Without this, stepping into a function would stop at the declaration
		// line with locals not yet visible.
		switch vm.prg.code[pc].(type) {
		case *enterFuncStashless, *enterFunc, *enterFuncBody:
			vm.prg.code[pc].exec(vm)
			continue
		}

		// Statement boundary detection: check if source position changed
		if vm.prg != nil && vm.prg.src != nil {
			srcOffset := vm.prg.sourceOffset(pc)
			prgChanged := vm.prg != dbg.lastPrg
			if srcOffset != dbg.lastSrcOffset || prgChanged {
				dbg.lastSrcOffset = srcOffset
				dbg.lastPrg = vm.prg
				// Resolve actual line number to avoid firing multiple times
				// for the same statement (which can have multiple srcMap entries)
				line := vm.prg.src.Position(srcOffset).Line
				if line != dbg.lastLine || prgChanged {
					dbg.lastLine = line
					// At a new statement — check breakpoints and stepping
					if event, bp, shouldPause := dbg.shouldPause(vm); shouldPause {
						dbg.invokeHook(vm, event)
					} else if bp != nil && bp.LogMessage != "" {
						// Log point: evaluate message and notify, don't pause
						msg := dbg.evalLogMessage(vm, bp.LogMessage)
						if dbg.logHook != nil {
							dbg.logHook(msg, vm.currentPosition())
						}
					}
				}
			}
		}

		vm.prg.code[pc].exec(vm)
	}

	if interrupted {
		vm.interruptLock.Lock()
		v := &InterruptedError{
			iface: vm.interruptVal,
		}
		v.stack = vm.captureStack(nil, 0)
		vm.interruptLock.Unlock()
		panic(v)
	}
}

// exceptionIsCaught checks if any try frame on the stack has a catch handler.
func (vm *vm) exceptionIsCaught() bool {
	for i := len(vm.tryStack) - 1; i >= 0; i-- {
		tf := &vm.tryStack[i]
		if tf.catchPos >= 0 && tf.catchPos != tryPanicMarker {
			return true
		}
	}
	return false
}

// dbgSaveCtx pushes a debug frame onto the debugger's parallel call stack.
func (vm *vm) dbgSaveCtx() {
	dbg := vm.dbg
	dbg.frames = append(dbg.frames, debugFrame{
		scopeLen:   len(vm.dbgScopes),
		lastLine:   dbg.lastLine,
		lastOffset: dbg.lastSrcOffset,
		lastPrg:    dbg.lastPrg,
	})
}

// dbgRestoreCtx pops the topmost debug frame and restores state.
func (vm *vm) dbgRestoreCtx() {
	dbg := vm.dbg
	n := len(dbg.frames)
	if n == 0 {
		return
	}
	f := dbg.frames[n-1]
	dbg.frames = dbg.frames[:n-1]

	vm.dbgScopes = vm.dbgScopes[:f.scopeLen]
	if dbg.stepAction == DebugContinue {
		// Restore debug tracking to prevent breakpoint double-fire
		// when returning to the same line (e.g., after a function call
		// on a line with a conditional breakpoint).
		dbg.lastLine = f.lastLine
		dbg.lastSrcOffset = f.lastOffset
		dbg.lastPrg = f.lastPrg
	} else {
		// During stepping, force statement boundary detection so
		// step-over/step-out correctly pause after returning from a call.
		dbg.lastLine = -1
		dbg.lastSrcOffset = -1
		dbg.lastPrg = nil
	}
}

// dbgUnwind truncates the debugger's frame stack and scope stack to match a
// call stack unwound to callStackLen (used during exception propagation).
func (vm *vm) dbgUnwind(callStackLen int) {
	dbg := vm.dbg
	if callStackLen < len(dbg.frames) {
		scopeLen := 0
		if callStackLen > 0 {
			scopeLen = dbg.frames[callStackLen-1].scopeLen
		}
		dbg.frames = dbg.frames[:callStackLen]
		if scopeLen < len(vm.dbgScopes) {
			vm.dbgScopes = vm.dbgScopes[:scopeLen]
		}
	}
}

// dbgCheckException checks exception breakpoint conditions and fires the debug
// hook if appropriate. Called from handleThrow.
func (vm *vm) dbgCheckException(ex *Exception) {
	dbg := vm.dbg
	// Deduplicate: handleThrow can be called multiple times for the same
	// exception during propagation (e.g., from _throw.exec and vm.try recovery).
	if ex != dbg.lastException {
		caught := vm.exceptionIsCaught()
		if dbg.exFilterAll || (dbg.exFilterUncaught && !caught) {
			dbg.lastException = ex
			dbg.currentException = ex
			dbg.invokeHook(vm, DebugEventException)
			dbg.currentException = nil
		}
	}
}

// dbgPushBlockScope pushes debug scope info for a block's stack-register variables.
func (vm *vm) dbgPushBlockScope(dbgNames map[unistring.String]int, sp int) {
	vars := make(map[unistring.String]int, len(dbgNames))
	for name, offset := range dbgNames {
		vars[name] = sp + offset
	}
	vm.dbgScopes = append(vm.dbgScopes, dbgScopeInfo{vars: vars})
}

// dbgPushFuncScope pushes debug scope info for a function's stack-register variables.
// Arguments are encoded with negative offsets: -(argIdx+1).
func (vm *vm) dbgPushFuncScope(dbgNames map[unistring.String]int, sb, args int) {
	vars := make(map[unistring.String]int, len(dbgNames))
	for name, offset := range dbgNames {
		if offset < 0 {
			// Arg: -(argIdx+1), actual position = sb + 1 + argIdx
			vars[name] = sb + 1 + (-offset - 1)
		} else {
			// Local: absolute position = sb + args + 1 + offset
			vars[name] = sb + args + 1 + offset
		}
	}
	vm.dbgScopes = append(vm.dbgScopes, dbgScopeInfo{vars: vars})
}

// dbgPopScope pops the topmost debug scope entry.
func (vm *vm) dbgPopScope() {
	if n := len(vm.dbgScopes); n > 0 {
		vm.dbgScopes = vm.dbgScopes[:n-1]
	}
}
