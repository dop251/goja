package goja

import (
	"sync"
	"testing"
	"time"
)

func TestDebuggerBreakpoint(t *testing.T) {
	const SCRIPT = `
	function test() {
		var a = 1;
		var b = 2;
		var c = a + b; // line 5
		return c;
	}
	test();
	`

	vm := New()
	debugger := vm.EnableDebugger()

	var hitBreakpoint bool
	var stateCapture *DebuggerState
	var wg sync.WaitGroup
	wg.Add(1)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		hitBreakpoint = true
		stateCapture = state
		wg.Done()
		return DebugContinue
	})

	// Add breakpoint at line 5
	debugger.AddBreakpoint("", 5, 0)

	// Run in goroutine because debugger will pause
	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	// Wait for breakpoint or timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for breakpoint")
	}

	if !hitBreakpoint {
		t.Error("Breakpoint was not hit")
	}

	if stateCapture == nil {
		t.Fatal("State was not captured")
	}

	if stateCapture.SourcePos.Line != 5 {
		t.Errorf("Expected breakpoint at line 5, got line %d", stateCapture.SourcePos.Line)
	}
}

func TestDebuggerStepInto(t *testing.T) {
	const SCRIPT = `
	function add(x, y) {
		return x + y;
	}
	
	function main() {
		var result = add(1, 2);
		return result;
	}
	
	main();
	`

	vm := New()
	debugger := vm.EnableDebugger()

	var steps []int
	var mu sync.Mutex
	stepDone := make(chan bool)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		mu.Lock()
		steps = append(steps, state.SourcePos.Line)
		mu.Unlock()

		if len(steps) >= 5 {
			stepDone <- true
			return DebugContinue
		}
		return DebugStepInto
	})

	// Enable step mode from the start
	debugger.SetStepMode(true)

	// Run in goroutine
	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	select {
	case <-stepDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout during stepping")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(steps) < 3 {
		t.Errorf("Expected at least 3 steps, got %d", len(steps))
	}
}

func TestDebuggerStepOver(t *testing.T) {
	const SCRIPT = `
	function helper() {
		var x = 1;
		var y = 2;
		return x + y;
	}
	
	var a = 10;
	var b = helper(); // Should step over this
	var c = a + b;
	`

	vm := New()
	debugger := vm.EnableDebugger()

	var lines []int
	var mu sync.Mutex
	done := make(chan bool)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		mu.Lock()
		lines = append(lines, state.SourcePos.Line)
		mu.Unlock()

		if state.SourcePos.Line == 10 { // After the function call
			done <- true
			return DebugContinue
		}

		if state.SourcePos.Line == 8 { // At the function call line
			return DebugStepOver
		}

		return DebugStepInto
	})

	debugger.SetStepMode(true)

	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout during step over")
	}

	mu.Lock()
	defer mu.Unlock()

	// Check that we didn't step into the helper function
	for _, line := range lines {
		if line >= 2 && line <= 5 {
			t.Error("Stepped into helper function when should have stepped over")
		}
	}
}

func TestDebuggerMultipleBreakpoints(t *testing.T) {
	const SCRIPT = `
	var sum = 0;
	for (var i = 0; i < 3; i++) {
		sum += i; // line 3
	}
	var result = sum; // line 5
	`

	vm := New()
	debugger := vm.EnableDebugger()

	var breakpointHits []int
	var mu sync.Mutex
	done := make(chan bool)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		mu.Lock()
		if state.Breakpoint != nil {
			breakpointHits = append(breakpointHits, state.SourcePos.Line)
		}
		mu.Unlock()

		if state.SourcePos.Line == 5 {
			done <- true
		}
		return DebugContinue
	})

	// Add breakpoints
	bp1 := debugger.AddBreakpoint("", 3, 0)
	bp2 := debugger.AddBreakpoint("", 5, 0)

	// Check that breakpoints were added
	bps := debugger.GetBreakpoints()
	if len(bps) != 2 {
		t.Fatalf("Expected 2 breakpoints, got %d", len(bps))
	}

	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for breakpoints")
	}

	mu.Lock()
	defer mu.Unlock()

	// Should hit line 3 three times (loop) and line 5 once
	if len(breakpointHits) != 4 {
		t.Errorf("Expected 4 breakpoint hits, got %d", len(breakpointHits))
	}

	// Test disabling a breakpoint
	debugger.EnableBreakpoint(bp1, false)
	
	// Test removing a breakpoint
	if !debugger.RemoveBreakpoint(bp2) {
		t.Error("Failed to remove breakpoint")
	}
}

func TestDebuggerPauseResume(t *testing.T) {
	const SCRIPT = `
	var count = 0;
	while (count < 100) {
		count++;
	}
	`

	vm := New()
	debugger := vm.EnableDebugger()

	paused := make(chan bool)
	resumed := make(chan bool)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		paused <- true
		<-resumed
		return DebugContinue
	})

	// Run script in background
	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	// Give script time to start
	time.Sleep(10 * time.Millisecond)

	// Pause execution
	debugger.Pause()

	// Wait for pause
	select {
	case <-paused:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for pause")
	}

	// Resume execution
	resumed <- true
}

func TestDebuggerStepOut(t *testing.T) {
	const SCRIPT = `
	function inner() {
		var x = 1;
		var y = 2;
		return x + y; // line 4
	}
	
	function outer() {
		var a = inner();
		return a * 2; // line 9
	}
	
	outer();
	`

	vm := New()
	debugger := vm.EnableDebugger()

	hitLine9 := make(chan bool)

	debugger.SetHandler(func(state *DebuggerState) DebugCommand {
		if state.SourcePos.Line == 3 {
			// Inside inner function, step out
			return DebugStepOut
		}
		if state.SourcePos.Line == 9 {
			hitLine9 <- true
			return DebugContinue
		}
		return DebugStepInto
	})

	// Add breakpoint inside inner function
	debugger.AddBreakpoint("", 3, 0)

	go func() {
		_, err := vm.RunString(SCRIPT)
		if err != nil {
			t.Errorf("Script execution failed: %v", err)
		}
	}()

	select {
	case <-hitLine9:
		// Success - we stepped out to the outer function
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for step out")
	}
}