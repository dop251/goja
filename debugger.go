package goja

import (
	"sync"
)

// DebugFlags controls the debugging behavior
type DebugFlags uint32

const (
	// FlagStepMode enables step-by-step execution
	FlagStepMode DebugFlags = 1 << iota
	// FlagPaused indicates the VM is currently paused
	FlagPaused
)

// Position represents a position in source code
type Position struct {
	Filename string
	Line     int
	Column   int
}

// Breakpoint represents a breakpoint in the code
type Breakpoint struct {
	id        int
	SourcePos Position // Position in source code
	pc        int      // Program counter position (-1 if not resolved)
	enabled   bool
	hit       int // Number of times this breakpoint was hit
}

// ID returns the breakpoint ID
func (b *Breakpoint) ID() int {
	return b.id
}

// DebuggerState represents the current state when paused
type DebuggerState struct {
	PC           int
	SourcePos    Position
	CallStack    []StackFrame
	Breakpoint   *Breakpoint // Current breakpoint if stopped at one
	StepMode     bool
}

// DebugHandler is called when the debugger pauses execution
type DebugHandler func(state *DebuggerState) DebugCommand

// DebugCommand represents commands that can be sent to the debugger
type DebugCommand int

const (
	DebugContinue DebugCommand = iota
	DebugStepOver
	DebugStepInto
	DebugStepOut
	DebugPause
)

// Debugger provides debugging capabilities for the Runtime
type Debugger struct {
	mu          sync.RWMutex
	runtime     *Runtime
	breakpoints map[int]*Breakpoint
	nextID      int
	flags       DebugFlags
	handler     DebugHandler
	
	// Internal state
	pcBreakpoints map[int]*Breakpoint // PC to breakpoint mapping for fast lookup
	stepDepth     int                 // Call stack depth for step over/out
	stepMode      DebugCommand
}

// NewDebugger creates a new debugger for the runtime
func (r *Runtime) NewDebugger() *Debugger {
	return &Debugger{
		runtime:       r,
		breakpoints:   make(map[int]*Breakpoint),
		pcBreakpoints: make(map[int]*Breakpoint),
	}
}

// SetHandler sets the debug handler that will be called when execution pauses
func (d *Debugger) SetHandler(handler DebugHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
}

// AddBreakpoint adds a breakpoint at the specified source position
func (d *Debugger) AddBreakpoint(filename string, line, column int) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	bp := &Breakpoint{
		id:      d.nextID,
		SourcePos: Position{
			Filename: filename,
			Line:     line,
			Column:   column,
		},
		pc:      -1,
		enabled: true,
	}
	
	d.nextID++
	d.breakpoints[bp.id] = bp
	
	// Try to resolve the breakpoint to a PC if we have a program loaded
	if d.runtime.vm != nil && d.runtime.vm.prg != nil {
		d.resolveBreakpoint(bp)
	}
	
	return bp.id
}

// RemoveBreakpoint removes a breakpoint by ID
func (d *Debugger) RemoveBreakpoint(id int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	bp, exists := d.breakpoints[id]
	if !exists {
		return false
	}
	
	delete(d.breakpoints, id)
	if bp.pc >= 0 {
		delete(d.pcBreakpoints, bp.pc)
	}
	
	return true
}

// EnableBreakpoint enables or disables a breakpoint
func (d *Debugger) EnableBreakpoint(id int, enabled bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	bp, exists := d.breakpoints[id]
	if !exists {
		return false
	}
	
	bp.enabled = enabled
	return true
}

// GetBreakpoints returns all breakpoints
func (d *Debugger) GetBreakpoints() []*Breakpoint {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	result := make([]*Breakpoint, 0, len(d.breakpoints))
	for _, bp := range d.breakpoints {
		result = append(result, bp)
	}
	
	return result
}

// SetStepMode enables or disables step mode
func (d *Debugger) SetStepMode(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if enabled {
		d.flags |= FlagStepMode
		d.stepMode = DebugStepInto // Default to step into
	} else {
		d.flags &^= FlagStepMode
	}
}

// Continue resumes execution
func (d *Debugger) Continue() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags &^= FlagPaused
	d.stepMode = DebugContinue
}

// StepOver executes the next line, stepping over function calls
func (d *Debugger) StepOver() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepOver
	d.stepDepth = len(d.runtime.vm.callStack)
}

// StepInto executes the next line, stepping into function calls
func (d *Debugger) StepInto() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagStepMode
	d.flags &^= FlagPaused
	d.stepMode = DebugStepInto
}

// StepOut continues execution until the current function returns
func (d *Debugger) StepOut() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags &^= FlagPaused
	d.stepMode = DebugStepOut
	d.stepDepth = len(d.runtime.vm.callStack) - 1
}

// Pause pauses execution at the next opportunity
func (d *Debugger) Pause() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.flags |= FlagPaused
}

// resolveBreakpoint tries to resolve a source position to a PC
func (d *Debugger) resolveBreakpoint(bp *Breakpoint) {
	prg := d.runtime.vm.prg
	if prg == nil || prg.src == nil {
		return
	}
	
	// Find the PC that corresponds to this source position
	// We need to search through srcMap items
	for pc := 0; pc < len(prg.code); pc++ {
		if pc < len(prg.srcMap) {
			item := prg.srcMap[pc]
			if item.srcPos >= 0 {
				pos := prg.src.Position(item.srcPos)
				if pos.Filename == bp.SourcePos.Filename &&
					pos.Line == bp.SourcePos.Line {
					// Found a match - we accept any column on the same line
					bp.pc = pc
					d.pcBreakpoints[pc] = bp
					break
				}
			}
		}
	}
}

// resolvePendingBreakpoints resolves all breakpoints that haven't been resolved yet
func (d *Debugger) resolvePendingBreakpoints() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for _, bp := range d.breakpoints {
		if bp.pc < 0 {
			d.resolveBreakpoint(bp)
		}
	}
}

// checkBreakpoint is called by the VM to check if we should pause
func (d *Debugger) checkBreakpoint(vm *vm) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Check if paused
	if d.flags&FlagPaused != 0 {
		return true
	}
	
	// Check breakpoints
	if bp, exists := d.pcBreakpoints[vm.pc]; exists && bp.enabled {
		bp.hit++
		d.flags |= FlagPaused
		return true
	}
	
	// Check step mode
	if d.flags&FlagStepMode != 0 {
		switch d.stepMode {
		case DebugStepInto:
			d.flags |= FlagPaused
			return true
		case DebugStepOver:
			if len(vm.callStack) <= d.stepDepth {
				d.flags |= FlagPaused
				return true
			}
		case DebugStepOut:
			if len(vm.callStack) < d.stepDepth {
				d.flags |= FlagPaused
				return true
			}
		}
	}
	
	return false
}

// handlePause is called when the VM pauses
func (d *Debugger) handlePause(vm *vm) {
	d.mu.RLock()
	handler := d.handler
	d.mu.RUnlock()
	
	if handler == nil {
		// No handler, just continue
		d.Continue()
		return
	}
	
	// Build debug state
	state := &DebuggerState{
		PC:        vm.pc,
		StepMode:  d.flags&FlagStepMode != 0,
	}
	
	// Get source position
	if vm.prg != nil && vm.prg.srcMap != nil && vm.pc < len(vm.prg.srcMap) {
		item := vm.prg.srcMap[vm.pc]
		if item.srcPos >= 0 && vm.prg.src != nil {
			pos := vm.prg.src.Position(item.srcPos)
			state.SourcePos = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}
	}
	
	// Get current breakpoint if any
	d.mu.RLock()
	if bp, exists := d.pcBreakpoints[vm.pc]; exists {
		state.Breakpoint = bp
	}
	d.mu.RUnlock()
	
	// Capture call stack
	state.CallStack = d.runtime.CaptureCallStack(0, nil)
	
	// Call handler and process command
	cmd := handler(state)
	switch cmd {
	case DebugContinue:
		d.Continue()
	case DebugStepOver:
		d.StepOver()
	case DebugStepInto:
		d.StepInto()
	case DebugStepOut:
		d.StepOut()
	case DebugPause:
		// Already paused, do nothing
	}
}