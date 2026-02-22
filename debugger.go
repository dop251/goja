package goja

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dop251/goja/unistring"
)

// DebugAction tells the VM what to do after a debug hook returns.
type DebugAction int

const (
	// DebugContinue resumes normal execution.
	DebugContinue DebugAction = iota
	// DebugStepOver executes the next statement at the same or shallower call depth.
	DebugStepOver
	// DebugStepIn executes the next statement, stepping into function calls.
	DebugStepIn
	// DebugStepOut resumes until the current function returns.
	DebugStepOut
)

// DebugEvent describes why the debugger paused.
type DebugEvent int

const (
	// DebugEventBreakpoint indicates execution hit a breakpoint.
	DebugEventBreakpoint DebugEvent = iota
	// DebugEventStep indicates a step operation completed.
	DebugEventStep
	// DebugEventPause indicates a user-requested pause.
	DebugEventPause
	// DebugEventDebuggerStmt indicates a `debugger` statement was hit.
	DebugEventDebuggerStmt
	// DebugEventEntry indicates the program entry point (stopOnEntry).
	DebugEventEntry
	// DebugEventException indicates an exception was thrown.
	DebugEventException
)

// DebugPosition represents a resolved source location.
type DebugPosition struct {
	Filename string
	Line     int
	Column   int
}

// DebugVariable represents a variable visible in a scope.
type DebugVariable struct {
	Name  string
	Value Value
}

// DebugScope represents a variable scope for inspection.
type DebugScope struct {
	Type      string // "local", "closure", "block", "global", "with"
	Name      string // Human-readable scope name
	Variables []DebugVariable
}

// Breakpoint represents a breakpoint set in a source file.
type Breakpoint struct {
	ID           int
	Filename     string // Canonical path
	Line         int
	Column       int    // 0 means any column
	Condition    string // JS expression; pause only when truthy
	HitCondition string // e.g., ">5", "==3", "10" (every 10th hit)
	LogMessage   string // If set, log instead of pausing
	hitCount     int    // internal counter
}

// BreakpointOption configures optional breakpoint properties.
type BreakpointOption func(*Breakpoint)

// WithCondition sets a conditional expression on a breakpoint.
// The breakpoint only fires when the expression evaluates to a truthy value.
func WithCondition(expr string) BreakpointOption {
	return func(bp *Breakpoint) {
		bp.Condition = expr
	}
}

// WithHitCondition sets a hit count condition on a breakpoint.
// Supported formats: "N" (every Nth hit), ">N", ">=N", "<N", "<=N", "==N", "!=N".
func WithHitCondition(expr string) BreakpointOption {
	return func(bp *Breakpoint) {
		bp.HitCondition = expr
	}
}

// WithLogMessage sets a log message on a breakpoint, turning it into a log point.
// When hit, the message is logged instead of pausing execution.
// Use {expression} for interpolation, e.g., "x = {x}, y = {y}".
func WithLogMessage(msg string) BreakpointOption {
	return func(bp *Breakpoint) {
		bp.LogMessage = msg
	}
}

// DebugContext provides safe access to VM state while the VM is paused.
// It is only valid for the duration of the debug hook call — do not retain references.
type DebugContext struct {
	vm *vm
}

// CallStack returns the current call stack.
func (dc *DebugContext) CallStack() []StackFrame {
	return dc.vm.r.CaptureCallStack(0, nil)
}

// Scopes returns variable scopes for the given stack frame index
// (0 = top/current frame, 1 = caller, etc).
func (dc *DebugContext) Scopes(frameIndex int) []DebugScope {
	return dc.vm.debugScopes(frameIndex)
}

// Eval evaluates an expression in the context of the given stack frame.
// Debug hooks are disabled during evaluation to prevent re-entry.
// The expression may have side effects (mutating eval).
func (dc *DebugContext) Eval(frameIndex int, expr string) (Value, error) {
	return dc.vm.debugEval(frameIndex, expr)
}

// Runtime returns the underlying Runtime.
func (dc *DebugContext) Runtime() *Runtime {
	return dc.vm.r
}

// Exception returns the current exception value when paused on an exception breakpoint.
// Returns nil if not paused on an exception.
func (dc *DebugContext) Exception() *Exception {
	if dc.vm.dbg != nil {
		return dc.vm.dbg.currentException
	}
	return nil
}

// SetVariable sets a variable's value in the specified scope.
// frameIndex selects the stack frame (0 = current), scopeIndex selects the scope
// within that frame (as returned by Scopes()). Returns an error if the variable
// is not found or cannot be set.
func (dc *DebugContext) SetVariable(frameIndex, scopeIndex int, name string, value Value) error {
	return dc.vm.debugSetVariable(frameIndex, scopeIndex, name, value)
}

// DebugHookFunc is called at statement boundaries when the debugger decides to pause.
// It receives a DebugContext for inspecting VM state, the event type, and the current position.
// It must return the desired next action.
// This function is called on the goroutine that runs the VM — do not block indefinitely
// without providing a way to resume (e.g., channel-based coordination).
type DebugHookFunc func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction

// DebugLogFunc is called when a log point fires. It receives the evaluated message
// and the source position. Called on the VM goroutine.
type DebugLogFunc func(message string, pos DebugPosition)

// debugFrame stores per-call-frame debug state on a parallel stack inside the
// Debugger. This keeps the context struct (which exists for every call frame
// even when not debugging) free of debug fields.
type debugFrame struct {
	scopeLen   int      // len(vm.dbgScopes) at the time of the call
	lastLine   int      // debugger lastLine at the time of the call
	lastOffset int      // debugger lastSrcOffset at the time of the call
	lastPrg    *Program // debugger lastPrg at the time of the call
}

// Debugger holds all debug state for a runtime.
type Debugger struct {
	hook    DebugHookFunc
	logHook DebugLogFunc

	// Breakpoint management — protected by bpMu
	bpMu        sync.RWMutex
	breakpoints map[int]*Breakpoint              // id -> Breakpoint
	nextBpID    int                               //nolint:unused
	bpIndex     map[string]map[int][]*Breakpoint // canonical filename -> line -> breakpoints
	bpByBase    map[string]string                 // basename -> canonical path (for cross-resolution)
	bpCount     int32                             // atomic; fast-path: skip map lookup when 0

	// Exception breakpoints — protected by bpMu
	exFilterAll      bool // pause on all thrown exceptions
	exFilterUncaught bool // pause only on uncaught exceptions

	// Per-call-frame debug state — VM-goroutine owned.
	// Parallel to vm.callStack; pushed in saveCtx, popped in restoreCtx.
	frames []debugFrame

	// Stepping state — VM-goroutine owned (only accessed during execution)
	stepAction    DebugAction
	stepDepth     int
	lastSrcOffset int
	lastLine      int
	lastPrg       *Program

	// Exception state — VM-goroutine owned
	currentException *Exception // set during exception breakpoint hook
	lastException    *Exception // dedup: prevent double-fire during propagation

	// Cross-goroutine pause request
	pauseRequested uint32 // atomic
}

// NewDebugger creates a new Debugger with the given hook function.
// The hook is called on the VM goroutine at breakpoints, step completions,
// debugger statements, and pause requests.
func NewDebugger(hook DebugHookFunc) *Debugger {
	return &Debugger{
		hook:        hook,
		breakpoints: make(map[int]*Breakpoint),
		bpIndex:     make(map[string]map[int][]*Breakpoint),
		bpByBase:    make(map[string]string),
	}
}

// SetLogHook sets the callback for log point messages.
// Called on the VM goroutine when a log point fires.
func (d *Debugger) SetLogHook(fn DebugLogFunc) {
	d.logHook = fn
}

// SetExceptionBreakpoints configures which exceptions should pause execution.
// Supported filter values: "all" (all exceptions), "uncaught" (uncaught only).
// Safe to call from any goroutine.
func (d *Debugger) SetExceptionBreakpoints(filters []string) {
	d.bpMu.Lock()
	defer d.bpMu.Unlock()

	d.exFilterAll = false
	d.exFilterUncaught = false
	for _, f := range filters {
		switch f {
		case "all":
			d.exFilterAll = true
		case "uncaught":
			d.exFilterUncaught = true
		}
	}
}

// HasExceptionBreakpoints returns true if any exception filter is active.
func (d *Debugger) HasExceptionBreakpoints() bool {
	d.bpMu.RLock()
	defer d.bpMu.RUnlock()
	return d.exFilterAll || d.exFilterUncaught
}

// SetBreakpoint adds a breakpoint at the given source location.
// The filename is canonicalized for consistent matching.
// Column 0 means any column on that line.
// Safe to call from any goroutine.
func (d *Debugger) SetBreakpoint(filename string, line, column int, opts ...BreakpointOption) *Breakpoint {
	canonical := canonicalizePath(filename)

	d.bpMu.Lock()
	defer d.bpMu.Unlock()

	d.nextBpID++
	bp := &Breakpoint{
		ID:       d.nextBpID,
		Filename: canonical,
		Line:     line,
		Column:   column,
	}
	for _, opt := range opts {
		opt(bp)
	}

	d.breakpoints[bp.ID] = bp

	lineMap, ok := d.bpIndex[canonical]
	if !ok {
		lineMap = make(map[int][]*Breakpoint)
		d.bpIndex[canonical] = lineMap
	}
	lineMap[line] = append(lineMap[line], bp)

	// Register basename → canonical mapping for cross-resolution.
	// This allows breakpoints set with full paths (e.g. from VS Code) to match
	// goja sources registered with short names (e.g. "fibonacci.ts"), and vice versa.
	base := filepath.Base(canonical)
	if _, exists := d.bpByBase[base]; !exists {
		d.bpByBase[base] = canonical
	}

	atomic.AddInt32(&d.bpCount, 1)
	return bp
}

// RemoveBreakpoint removes the breakpoint with the given ID.
// Safe to call from any goroutine.
func (d *Debugger) RemoveBreakpoint(id int) bool {
	d.bpMu.Lock()
	defer d.bpMu.Unlock()

	bp, ok := d.breakpoints[id]
	if !ok {
		return false
	}
	delete(d.breakpoints, id)

	if lineMap, ok := d.bpIndex[bp.Filename]; ok {
		bps := lineMap[bp.Line]
		for i, b := range bps {
			if b.ID == id {
				bps[i] = bps[len(bps)-1]
				bps[len(bps)-1] = nil
				lineMap[bp.Line] = bps[:len(bps)-1]
				break
			}
		}
		if len(lineMap[bp.Line]) == 0 {
			delete(lineMap, bp.Line)
		}
		if len(lineMap) == 0 {
			delete(d.bpIndex, bp.Filename)
		}
	}

	atomic.AddInt32(&d.bpCount, -1)
	return true
}

// ClearBreakpoints removes all breakpoints for the given filename.
// Safe to call from any goroutine.
func (d *Debugger) ClearBreakpoints(filename string) {
	canonical := canonicalizePath(filename)

	d.bpMu.Lock()
	defer d.bpMu.Unlock()

	lineMap, ok := d.bpIndex[canonical]
	if !ok {
		return
	}

	var count int32
	for _, bps := range lineMap {
		for _, bp := range bps {
			delete(d.breakpoints, bp.ID)
			count++
		}
	}
	delete(d.bpIndex, canonical)

	atomic.AddInt32(&d.bpCount, -count)
}

// GetBreakpoints returns a copy of all current breakpoints.
// Safe to call from any goroutine.
func (d *Debugger) GetBreakpoints() []*Breakpoint {
	d.bpMu.RLock()
	defer d.bpMu.RUnlock()

	result := make([]*Breakpoint, 0, len(d.breakpoints))
	for _, bp := range d.breakpoints {
		bpCopy := *bp
		result = append(result, &bpCopy)
	}
	return result
}

// shouldPause checks whether the VM should pause at the current position.
// Called on the VM goroutine at statement boundaries.
// Returns the event type, the matching breakpoint (if any), and whether to pause.
// For log points, returns the breakpoint but shouldPause=false.
func (d *Debugger) shouldPause(vm *vm) (DebugEvent, *Breakpoint, bool) {
	// Check breakpoints (fast path: skip if no breakpoints set)
	if atomic.LoadInt32(&d.bpCount) > 0 && vm.prg != nil && vm.prg.src != nil {
		pos := vm.prg.src.Position(vm.prg.sourceOffset(vm.pc))
		canonical := canonicalizePath(pos.Filename)

		d.bpMu.RLock()
		lineMap, ok := d.bpIndex[canonical]
		if !ok {
			// Fallback: try matching by basename (handles VS Code full paths
			// matching goja sources registered with short names, and vice versa).
			base := filepath.Base(canonical)
			if altPath, found := d.bpByBase[base]; found && altPath != canonical {
				lineMap, ok = d.bpIndex[altPath]
			}
		}
		if ok {
			if bps, ok := lineMap[pos.Line]; ok {
				for _, bp := range bps {
					if bp.Column == 0 || bp.Column == pos.Column {
						d.bpMu.RUnlock()

						// Increment hit count
						bp.hitCount++

						// Check hit condition
						if bp.HitCondition != "" && !evalHitCondition(bp.HitCondition, bp.hitCount) {
							return 0, nil, false
						}

						// Check condition expression
						if bp.Condition != "" {
							val, err := vm.debugEval(0, bp.Condition)
							if err != nil || !val.ToBoolean() {
								return 0, nil, false
							}
						}

						// Log point: don't pause, return the bp for logging
						if bp.LogMessage != "" {
							return DebugEventBreakpoint, bp, false
						}

						return DebugEventBreakpoint, bp, true
					}
				}
			}
		}
		d.bpMu.RUnlock()
	}

	// Check stepping
	depth := len(vm.callStack)
	switch d.stepAction {
	case DebugStepOver:
		if depth <= d.stepDepth {
			return DebugEventStep, nil, true
		}
	case DebugStepIn:
		return DebugEventStep, nil, true
	case DebugStepOut:
		if depth < d.stepDepth {
			return DebugEventStep, nil, true
		}
	}

	return 0, nil, false
}

// evalLogMessage evaluates a log point message, replacing {expr} with eval results.
var logInterpolationRegex = regexp.MustCompile(`\{([^}]+)\}`)

func (d *Debugger) evalLogMessage(vm *vm, msg string) string {
	return logInterpolationRegex.ReplaceAllStringFunc(msg, func(match string) string {
		expr := match[1 : len(match)-1] // strip { and }
		val, err := vm.debugEval(0, expr)
		if err != nil {
			return "{" + expr + "}"
		}
		return val.String()
	})
}

// evalHitCondition evaluates a hit condition string against a hit count.
// Supported formats: "N" (every Nth hit), ">N", ">=N", "<N", "<=N", "==N", "!=N".
func evalHitCondition(expr string, hitCount int) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true
	}

	// Try operator+number patterns
	for _, prefix := range []string{">=", "<=", "!=", "==", ">", "<"} {
		if strings.HasPrefix(expr, prefix) {
			numStr := strings.TrimSpace(expr[len(prefix):])
			n, err := strconv.Atoi(numStr)
			if err != nil {
				return true // invalid → treat as always true
			}
			switch prefix {
			case ">":
				return hitCount > n
			case ">=":
				return hitCount >= n
			case "<":
				return hitCount < n
			case "<=":
				return hitCount <= n
			case "==":
				return hitCount == n
			case "!=":
				return hitCount != n
			}
		}
	}

	// Plain number: every Nth hit (modulo)
	n, err := strconv.Atoi(expr)
	if err != nil || n <= 0 {
		return true // invalid → treat as always true
	}
	return hitCount%n == 0
}

// invokeHook calls the debug hook and processes the returned action.
func (d *Debugger) invokeHook(vm *vm, event DebugEvent) {
	pos := vm.currentPosition()
	ctx := &DebugContext{vm: vm}
	action := d.hook(ctx, event, pos)
	d.stepAction = action
	if action != DebugContinue {
		d.stepDepth = len(vm.callStack)
	}
}

// dbgScopeRange returns the [start, end) range of dbgScopes entries that
// belong to the given stack frame. This replaces direct reads from context.dbgScopeLen
// with reads from the debugger's parallel frame stack.
func (vm *vm) dbgScopeRange(frameIndex int) (start, end int) {
	if vm.dbg == nil || len(vm.dbgScopes) == 0 {
		return 0, 0
	}
	frames := vm.dbg.frames
	// frameIndex 0 = current frame, which is above the top of frames/callStack
	fEndIdx := len(frames) - frameIndex
	if fEndIdx >= len(frames) {
		end = len(vm.dbgScopes)
	} else if fEndIdx >= 0 {
		end = frames[fEndIdx].scopeLen
	}
	fStartIdx := fEndIdx - 1
	if fStartIdx >= 0 && fStartIdx < len(frames) {
		start = frames[fStartIdx].scopeLen
	}
	return
}

// debugScopes returns variable scopes for the given stack frame index.
func (vm *vm) debugScopes(frameIndex int) []DebugScope {
	var scopes []DebugScope

	// Enumerate stack-register variables from dbgScopes for this frame.
	if start, end := vm.dbgScopeRange(frameIndex); end > start {
		// Iterate in reverse (innermost scope first)
		for i := end - 1; i >= start; i-- {
			ds := vm.dbgScopes[i]
			scope := DebugScope{
				Type: "block",
				Name: "Block",
			}
			for name, stackIdx := range ds.vars {
				var val Value
				if stackIdx >= 0 && stackIdx < len(vm.stack) {
					val = nilSafe(vm.stack[stackIdx])
				}
				if val == nil {
					val = _undefined
				}
				scope.Variables = append(scope.Variables, DebugVariable{
					Name:  name.String(),
					Value: val,
				})
			}
			scopes = append(scopes, scope)
		}
	}

	// Determine which stash to start from based on frame index
	var s *stash
	if frameIndex == 0 {
		s = vm.stash
	} else {
		idx := len(vm.callStack) - frameIndex
		if idx >= 0 && idx < len(vm.callStack) {
			s = vm.callStack[idx].stash
		}
	}

	isLocal := true
	for s != nil {
		scope := DebugScope{}
		isGlobal := s == &vm.r.global.stash

		if isGlobal {
			scope.Type = "global"
			scope.Name = "Global"
		} else if s.obj != nil {
			scope.Type = "with"
			scope.Name = "With Block"
		} else if isLocal {
			scope.Type = "local"
			scope.Name = "Local"
			isLocal = false
		} else {
			scope.Type = "closure"
			scope.Name = "Closure"
		}

		// Enumerate named stash bindings (closures, let/const in global, etc.)
		if s.names != nil {
			for name, idx := range s.names {
				realIdx := idx &^ maskTyp
				var val Value
				if int(realIdx) < len(s.values) {
					val = s.values[realIdx]
				}
				if val == nil {
					val = _undefined
				}
				scope.Variables = append(scope.Variables, DebugVariable{
					Name:  name.String(),
					Value: val,
				})
			}
		}

		// For with-statement scopes, enumerate the object properties
		if s.obj != nil {
			for _, key := range s.obj.Keys() {
				scope.Variables = append(scope.Variables, DebugVariable{
					Name:  key,
					Value: s.obj.Get(key),
				})
			}
		}

		// For global scope, also enumerate global object properties
		// (global var/function declarations are properties of the global object)
		if isGlobal {
			globalObj := vm.r.globalObject
			if globalObj != nil {
				for _, key := range globalObj.Keys() {
					scope.Variables = append(scope.Variables, DebugVariable{
						Name:  key,
						Value: globalObj.Get(key),
					})
				}
			}
		}

		scopes = append(scopes, scope)
		s = s.outer
	}
	return scopes
}

// debugEval evaluates an expression in the context of the given stack frame.
// Debug hooks are disabled during evaluation to prevent re-entry.
func (vm *vm) debugEval(frameIndex int, expr string) (retVal Value, err error) {
	// Save and restore debug state via defer to guarantee consistency
	savedDbg := vm.dbg
	savedStash := vm.stash
	savedPrg := vm.prg
	savedPc := vm.pc
	savedSp := vm.sp
	savedSb := vm.sb
	savedArgs := vm.args
	savedResult := vm.result

	defer func() {
		if r := recover(); r != nil {
			if ex, ok := r.(*InterruptedError); ok {
				err = ex
			} else if ex, ok := r.(*Exception); ok {
				err = ex
			} else {
				panic(r) // re-panic for unexpected errors
			}
		}
		vm.dbg = savedDbg
		vm.stash = savedStash
		vm.prg = savedPrg
		vm.pc = savedPc
		vm.sp = savedSp
		vm.sb = savedSb
		vm.args = savedArgs
		vm.result = savedResult
	}()

	// Compute scope range BEFORE disabling debug hooks (dbgScopeRange reads vm.dbg)
	scopeStart, scopeEnd := vm.dbgScopeRange(frameIndex)

	// Disable debug hooks during eval
	vm.dbg = nil

	// Switch to target frame's scope if needed
	if frameIndex > 0 {
		idx := len(vm.callStack) - frameIndex
		if idx >= 0 && idx < len(vm.callStack) {
			vm.stash = vm.callStack[idx].stash
		}
	}

	// Inject stack-register variables into a temporary stash so that
	// loadDynamic (used by eval-compiled code) can find let/const variables
	// that are optimized to stack registers.
	if scopeEnd > scopeStart {
		tmpStash := &stash{
			names: make(map[unistring.String]uint32),
			outer: vm.stash,
		}
		start, end := scopeStart, scopeEnd

		// Add variables from innermost to outermost; inner wins on name collision.
		seen := make(map[unistring.String]bool)
		for i := end - 1; i >= start; i-- {
			ds := vm.dbgScopes[i]
			for name, stackIdx := range ds.vars {
				if seen[name] {
					continue
				}
				seen[name] = true
				var val Value
				if stackIdx >= 0 && stackIdx < len(vm.stack) {
					val = nilSafe(vm.stack[stackIdx])
				}
				if val == nil {
					val = _undefined
				}
				si := uint32(len(tmpStash.values))
				tmpStash.names[name] = si | maskVar
				tmpStash.values = append(tmpStash.values, val)
			}
		}
		if len(tmpStash.names) > 0 {
			vm.stash = tmpStash
		}
	}

	p, compileErr := vm.r.compile("<debug-eval>", expr, false, false, vm)
	if compileErr != nil {
		return nil, compileErr
	}

	vm.pushCtx()
	vm.prg = p
	vm.pc = 0
	vm.args = 0
	vm.result = _undefined
	funcObj := Value(_undefined)
	if sb := vm.sb; sb > 0 && sb <= len(vm.stack) {
		funcObj = vm.stack[sb-1]
	}
	vm.push(funcObj)
	vm.sb = vm.sp
	vm.push(nil) // this
	ex := vm.runTry()
	retVal = vm.result
	vm.popCtx()
	if ex != nil {
		return nil, ex
	}
	vm.sp -= 2
	return retVal, nil
}

// debugSetVariable sets a variable in the specified scope and frame.
func (vm *vm) debugSetVariable(frameIndex, scopeIndex int, name string, value Value) error {
	// Count how many dbgScope entries belong to this frame (they come first in scope list)
	start, end := vm.dbgScopeRange(frameIndex)
	dbgScopeCount := end - start
	if dbgScopeCount > 0 {

		// If the target scope is a dbgScope (stack-register variable)
		if scopeIndex < dbgScopeCount {
			// dbgScopes are listed innermost first (reverse order)
			dsIdx := end - 1 - scopeIndex
			if dsIdx >= start && dsIdx < end {
				ds := vm.dbgScopes[dsIdx]
				uname := unistring.NewFromString(name)
				if stackIdx, ok := ds.vars[uname]; ok {
					if stackIdx >= 0 && stackIdx < len(vm.stack) {
						vm.stack[stackIdx] = value
						return nil
					}
				}
				return fmt.Errorf("variable %q not found in block scope", name)
			}
		}
	}

	// Walk the stash chain to find the target scope (adjusted for dbgScope count)
	stashScopeIndex := scopeIndex - dbgScopeCount
	var s *stash
	if frameIndex == 0 {
		s = vm.stash
	} else {
		idx := len(vm.callStack) - frameIndex
		if idx >= 0 && idx < len(vm.callStack) {
			s = vm.callStack[idx].stash
		}
	}
	if s == nil {
		return fmt.Errorf("invalid frame index: %d", frameIndex)
	}

	// Skip to the requested scope index
	for i := 0; i < stashScopeIndex && s != nil; i++ {
		s = s.outer
	}
	if s == nil {
		return fmt.Errorf("invalid scope index: %d", scopeIndex)
	}

	isGlobal := s == &vm.r.global.stash

	// For with-statement scopes, set on the object
	if s.obj != nil {
		if s.obj.Get(name) != nil {
			return s.obj.Set(name, value)
		}
		return fmt.Errorf("variable %q not found in with scope", name)
	}

	// For named stash bindings
	if s.names != nil {
		if idx, ok := s.names[unistring.NewFromString(name)]; ok {
			realIdx := idx &^ maskTyp
			if int(realIdx) < len(s.values) {
				s.values[realIdx] = value
				return nil
			}
		}
	}

	// For global scope, try the global object
	if isGlobal {
		globalObj := vm.r.globalObject
		if globalObj != nil && globalObj.Get(name) != nil {
			return globalObj.Set(name, value)
		}
	}

	return fmt.Errorf("variable %q not found", name)
}

// canonicalizePath normalizes a filename for consistent breakpoint matching.
func canonicalizePath(filename string) string {
	if filename == "" {
		return filename
	}

	// URLs: leave as-is
	if strings.HasPrefix(filename, "http://") || strings.HasPrefix(filename, "https://") || strings.HasPrefix(filename, "data:") {
		return filename
	}

	// Virtual/module paths (e.g., <eval>, <module:foo>): leave as-is
	if strings.HasPrefix(filename, "<") {
		return filename
	}

	// Filesystem paths
	cleaned := filepath.Clean(filename)
	if !filepath.IsAbs(cleaned) {
		if abs, err := filepath.Abs(cleaned); err == nil {
			cleaned = abs
		}
	}

	// Windows case-insensitive normalization
	if runtime.GOOS == "windows" {
		cleaned = strings.ToLower(cleaned)
	}

	return cleaned
}
