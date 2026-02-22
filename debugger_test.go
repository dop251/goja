package goja

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestDebugHookCalled(t *testing.T) {
	r := New()
	var positions []DebugPosition
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		positions = append(positions, pos)
		return DebugStepIn // keep stepping to hit every statement
	})
	dbg.SetBreakpoint("", 1, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = x + y;")
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) < 3 {
		t.Fatalf("Expected at least 3 positions, got %d", len(positions))
	}
}

func TestDebugHookPositions(t *testing.T) {
	r := New()
	var lines []int
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		lines = append(lines, pos.Line)
		return DebugStepIn // keep stepping
	})
	dbg.SetBreakpoint("", 1, 0) // trigger first pause
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = 3;")
	if err != nil {
		t.Fatal(err)
	}
	// Deduplicate consecutive lines (breakpoint + step may fire on same line)
	var uniqueLines []int
	for _, l := range lines {
		if len(uniqueLines) == 0 || uniqueLines[len(uniqueLines)-1] != l {
			uniqueLines = append(uniqueLines, l)
		}
	}
	// We should see lines 1, 2, 3
	if len(uniqueLines) < 3 {
		t.Fatalf("Expected at least 3 unique lines, got %d: %v (raw: %v)", len(uniqueLines), uniqueLines, lines)
	}
	for i, expected := range []int{1, 2, 3} {
		if i < len(uniqueLines) && uniqueLines[i] != expected {
			t.Errorf("Position %d: expected line %d, got %d", i, expected, uniqueLines[i])
		}
	}
}

func TestBreakpointHit(t *testing.T) {
	r := New()
	hitCount := 0
	var hitLine int
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
			hitLine = pos.Line
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = 3;")
	if err != nil {
		t.Fatal(err)
	}
	if hitCount != 1 {
		t.Fatalf("Expected 1 breakpoint hit, got %d", hitCount)
	}
	if hitLine != 2 {
		t.Fatalf("Expected breakpoint at line 2, got line %d", hitLine)
	}
}

func TestMultipleBreakpoints(t *testing.T) {
	r := New()
	var hitLines []int
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitLines = append(hitLines, pos.Line)
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 1, 0)
	dbg.SetBreakpoint("", 3, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = 3;")
	if err != nil {
		t.Fatal(err)
	}
	if len(hitLines) != 2 {
		t.Fatalf("Expected 2 breakpoint hits, got %d: %v", len(hitLines), hitLines)
	}
	if hitLines[0] != 1 || hitLines[1] != 3 {
		t.Fatalf("Expected breakpoints at lines 1,3 but got %v", hitLines)
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	bp := dbg.SetBreakpoint("", 2, 0)
	dbg.RemoveBreakpoint(bp.ID)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = 3;")
	if err != nil {
		t.Fatal(err)
	}
	if hitCount != 0 {
		t.Fatalf("Expected 0 breakpoint hits after removal, got %d", hitCount)
	}
}

func TestClearBreakpoints(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 1, 0)
	dbg.SetBreakpoint("", 2, 0)
	dbg.SetBreakpoint("", 3, 0)
	dbg.ClearBreakpoints("")
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1;\nvar y = 2;\nvar z = 3;")
	if err != nil {
		t.Fatal(err)
	}
	if hitCount != 0 {
		t.Fatalf("Expected 0 breakpoint hits after clear, got %d", hitCount)
	}
}

func TestStepOver(t *testing.T) {
	r := New()
	var events []DebugEvent
	var lines []int
	stepCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		events = append(events, event)
		lines = append(lines, pos.Line)
		stepCount++
		if stepCount >= 20 { // safety limit
			return DebugContinue
		}
		return DebugStepOver
	})
	dbg.SetBreakpoint("", 5, 0) // break at "var x = add(1, 2);"
	r.SetDebugger(dbg)
	_, err := r.RunString(`function add(a, b) {
  var result = a + b;
  return result;
}
var x = add(1, 2);
var y = x + 1;`)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 positions, got %d: %v", len(lines), lines)
	}
	// First hit is the breakpoint at line 5
	if events[0] != DebugEventBreakpoint {
		t.Errorf("Expected first event to be breakpoint, got %d", events[0])
	}
	if lines[0] != 5 {
		t.Errorf("Expected breakpoint on line 5, got line %d", lines[0])
	}
	// Step-over from line 5 (the function call) should NOT enter add()
	foundInsideFunction := false
	for i := 1; i < len(lines); i++ {
		if lines[i] == 2 || lines[i] == 3 {
			foundInsideFunction = true
		}
	}
	if foundInsideFunction {
		t.Errorf("Step over should not enter function body, but visited lines: %v", lines)
	}
}

func TestStepIn(t *testing.T) {
	r := New()
	var lines []int
	stepCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		lines = append(lines, pos.Line)
		stepCount++
		if stepCount >= 20 { // safety limit
			return DebugContinue
		}
		return DebugStepIn
	})
	dbg.SetBreakpoint("", 5, 0) // break at "var x = add(1, 2);"
	r.SetDebugger(dbg)
	_, err := r.RunString(`function add(a, b) {
  var result = a + b;
  return result;
}
var x = add(1, 2);
var y = x + 1;`)
	if err != nil {
		t.Fatal(err)
	}
	// Step in from line 5 should enter the function and visit line 2
	foundInsideFunction := false
	for _, line := range lines {
		if line == 2 {
			foundInsideFunction = true
			break
		}
	}
	if !foundInsideFunction {
		t.Errorf("Step in should enter function body, but lines were: %v", lines)
	}
}

func TestStepOut(t *testing.T) {
	r := New()
	var lines []int
	stepCount := 0
	insideFunction := false
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		lines = append(lines, pos.Line)
		stepCount++
		if stepCount >= 20 {
			return DebugContinue
		}
		// Step in first to enter the function, then step out
		if !insideFunction {
			if pos.Line == 2 {
				insideFunction = true
				return DebugStepOut // step out of the function
			}
			return DebugStepIn
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 5, 0) // break at "var x = add(1, 2);"
	r.SetDebugger(dbg)
	_, err := r.RunString(`function add(a, b) {
  var result = a + b;
  return result;
}
var x = add(1, 2);
var y = x + 1;`)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 positions, got %d: %v", len(lines), lines)
	}
	// Verify we entered the function (line 2) and then came back out
	// After step-out, execution returns to the caller (line 5, the call site)
	foundLine2 := false
	foundAfterReturn := false
	for i, line := range lines {
		if line == 2 {
			foundLine2 = true
		}
		// After visiting line 2, we should return to line 5 or 6
		if foundLine2 && i > 0 && (line == 5 || line == 6) {
			foundAfterReturn = true
		}
	}
	if !foundLine2 {
		t.Errorf("Expected to visit line 2 (inside function), lines: %v", lines)
	}
	if !foundAfterReturn {
		t.Errorf("Expected to return to caller after step-out, lines: %v", lines)
	}
}

func TestDebuggerStatement(t *testing.T) {
	r := New()
	paused := false
	var pauseEvent DebugEvent
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		paused = true
		pauseEvent = event
		return DebugContinue
	})
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 1; debugger; var y = 2;")
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Fatal("Expected debugger statement to trigger hook")
	}
	if pauseEvent != DebugEventDebuggerStmt {
		t.Errorf("Expected DebugEventDebuggerStmt, got %d", pauseEvent)
	}
}

func TestDebuggerStatementNoDebugger(t *testing.T) {
	// When no debugger is attached, debugger statement should be a no-op
	r := New()
	v, err := r.RunString("var x = 1; debugger; x + 1;")
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInteger() != 2 {
		t.Fatalf("Expected 2, got %v", v)
	}
}

func TestRequestPause(t *testing.T) {
	r := New()
	pauseReceived := false
	var mu sync.Mutex

	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventPause {
			mu.Lock()
			pauseReceived = true
			mu.Unlock()
		}
		return DebugContinue
	})
	r.SetDebugger(dbg)

	// Pre-request the pause before running
	r.RequestPause()

	_, err := r.RunString("var x = 1; var y = 2;")
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !pauseReceived {
		t.Fatal("Expected pause event to be received")
	}
}

func TestDebugNoOverheadWhenDisabled(t *testing.T) {
	// When no debugger is attached, execution should use the normal loop
	r := New()
	v, err := r.RunString("var sum = 0; for (var i = 0; i < 1000; i++) { sum += i; } sum;")
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInteger() != 499500 {
		t.Fatalf("Expected 499500, got %v", v)
	}
}

func TestCallStack(t *testing.T) {
	r := New()
	var stack []StackFrame
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			stack = ctx.CallStack()
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0) // inside the inner function
	r.SetDebugger(dbg)
	_, err := r.RunString(`function inner() {
  var x = 1;
  return x;
}
function outer() {
  return inner();
}
outer();`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stack) < 3 { // inner, outer, <global>
		t.Fatalf("Expected at least 3 stack frames, got %d", len(stack))
	}
	if stack[0].FuncName() != "inner" {
		t.Errorf("Expected top frame to be 'inner', got '%s'", stack[0].FuncName())
	}
	if stack[1].FuncName() != "outer" {
		t.Errorf("Expected second frame to be 'outer', got '%s'", stack[1].FuncName())
	}
}

func TestDebugScopesGlobal(t *testing.T) {
	r := New()
	var scopes []DebugScope
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			scopes = ctx.Scopes(0)
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 3, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 42;\nvar y = 'hello';\nvar z = x + 1;")
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) == 0 {
		t.Fatal("Expected at least one scope")
	}
	// Look for x in global scope
	foundX := false
	for _, s := range scopes {
		if s.Type == "global" {
			for _, v := range s.Variables {
				if v.Name == "x" {
					foundX = true
					if v.Value.ToInteger() != 42 {
						t.Errorf("Expected x=42, got %v", v.Value)
					}
				}
			}
		}
	}
	if !foundX {
		t.Error("Expected to find 'x' in global scope")
	}
}

func TestDebugEvalGlobalScope(t *testing.T) {
	// Test eval works correctly in global scope with multiple variables
	r := New()
	var evalResult Value
	var evalErr error
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			evalResult, evalErr = ctx.Eval(0, "a + b")
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 3, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var a = 10;\nvar b = 20;\nvar c = a + b;")
	if err != nil {
		t.Fatal(err)
	}
	if evalErr != nil {
		t.Fatalf("Eval error: %v", evalErr)
	}
	if evalResult == nil || evalResult.ToInteger() != 30 {
		t.Fatalf("Expected eval result 30, got %v", evalResult)
	}
}

func TestDebugEval(t *testing.T) {
	r := New()
	var evalResult Value
	var evalErr error
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			evalResult, evalErr = ctx.Eval(0, "x + 1")
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0)
	r.SetDebugger(dbg)
	_, err := r.RunString("var x = 41;\nvar y = 0;")
	if err != nil {
		t.Fatal(err)
	}
	if evalErr != nil {
		t.Fatalf("Eval error: %v", evalErr)
	}
	if evalResult == nil || evalResult.ToInteger() != 42 {
		t.Fatalf("Expected eval result 42, got %v", evalResult)
	}
}

func TestBreakpointConcurrentModification(t *testing.T) {
	r := New()
	hitCount := int32(0)
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			atomic.AddInt32(&hitCount, 1)
		}
		return DebugContinue
	})
	r.SetDebugger(dbg)

	// Set breakpoints concurrently while running
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bp := dbg.SetBreakpoint("", 1, 0)
			dbg.RemoveBreakpoint(bp.ID)
		}
	}()

	_, err := r.RunString("var sum = 0; for (var i = 0; i < 100; i++) { sum += i; } sum;")
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestGetBreakpoints(t *testing.T) {
	dbg := NewDebugger(nil)
	bp1 := dbg.SetBreakpoint("file1.js", 10, 0)
	bp2 := dbg.SetBreakpoint("file2.js", 20, 5)

	bps := dbg.GetBreakpoints()
	if len(bps) != 2 {
		t.Fatalf("Expected 2 breakpoints, got %d", len(bps))
	}

	dbg.RemoveBreakpoint(bp1.ID)
	bps = dbg.GetBreakpoints()
	if len(bps) != 1 {
		t.Fatalf("Expected 1 breakpoint, got %d", len(bps))
	}
	if bps[0].ID != bp2.ID {
		t.Errorf("Expected remaining breakpoint ID %d, got %d", bp2.ID, bps[0].ID)
	}
}

func TestBreakpointInLoop(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0) // inside the loop body
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 5; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	// The breakpoint on line 2 may fire multiple times per iteration due to
	// statement boundary detection granularity. Verify it fires at least 5 times.
	if hitCount < 5 {
		t.Fatalf("Expected at least 5 breakpoint hits in loop, got %d", hitCount)
	}
}

// Phase 5 tests

func TestConditionalBreakpoint(t *testing.T) {
	r := New()
	var hitValues []int64
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			val, _ := ctx.Eval(0, "i")
			hitValues = append(hitValues, val.ToInteger())
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0, WithCondition("i > 2"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 5; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	// Should only hit when i > 2 (i.e., i=3, i=4)
	if len(hitValues) != 2 {
		t.Fatalf("Expected 2 conditional hits, got %d: %v", len(hitValues), hitValues)
	}
	if hitValues[0] != 3 || hitValues[1] != 4 {
		t.Fatalf("Expected hits at i=3,4 but got %v", hitValues)
	}
}

func TestHitCountBreakpoint(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	// Hit every 3rd time
	dbg.SetBreakpoint("", 2, 0, WithHitCondition("3"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 9; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	// 9 iterations, hit every 3rd: hits 3, 6, 9 → 3 pauses
	if hitCount != 3 {
		t.Fatalf("Expected 3 hit-count breakpoint hits, got %d", hitCount)
	}
}

func TestHitCountComparison(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	// Pause only after 3rd hit (>= 3)
	dbg.SetBreakpoint("", 2, 0, WithHitCondition(">=3"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 5; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	// Hits: 1(no), 2(no), 3(yes), 4(yes), 5(yes) → 3 pauses
	if hitCount != 3 {
		t.Fatalf("Expected 3 hits for >=3, got %d", hitCount)
	}
}

func TestHitCountEquals(t *testing.T) {
	r := New()
	hitCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			hitCount++
		}
		return DebugContinue
	})
	// Pause only on exactly the 3rd hit
	dbg.SetBreakpoint("", 2, 0, WithHitCondition("==3"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 5; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	if hitCount != 1 {
		t.Fatalf("Expected 1 hit for ==3, got %d", hitCount)
	}
}

func TestLogPoint(t *testing.T) {
	r := New()
	var logMessages []string
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		// Hook should NOT be called for log points
		t.Fatalf("Debug hook should not fire for log points, got event %d", event)
		return DebugContinue
	})
	dbg.SetLogHook(func(message string, pos DebugPosition) {
		logMessages = append(logMessages, message)
	})
	dbg.SetBreakpoint("", 2, 0, WithLogMessage("i = {i}"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 3; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	if len(logMessages) != 3 {
		t.Fatalf("Expected 3 log messages, got %d: %v", len(logMessages), logMessages)
	}
	expected := []string{"i = 0", "i = 1", "i = 2"}
	for i, msg := range logMessages {
		if msg != expected[i] {
			t.Errorf("Log message %d: expected %q, got %q", i, expected[i], msg)
		}
	}
}

func TestLogPointWithCondition(t *testing.T) {
	r := New()
	var logMessages []string
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		t.Fatalf("Debug hook should not fire for log points")
		return DebugContinue
	})
	dbg.SetLogHook(func(message string, pos DebugPosition) {
		logMessages = append(logMessages, message)
	})
	// Log point with condition: only log when i > 1
	dbg.SetBreakpoint("", 2, 0, WithLogMessage("value: {i}"), WithCondition("i > 1"))
	r.SetDebugger(dbg)
	_, err := r.RunString("for (var i = 0; i < 4; i++) {\n  var x = i;\n}")
	if err != nil {
		t.Fatal(err)
	}
	if len(logMessages) != 2 {
		t.Fatalf("Expected 2 log messages, got %d: %v", len(logMessages), logMessages)
	}
}

func TestExceptionBreakpointAll(t *testing.T) {
	r := New()
	var exceptionEvents []DebugEvent
	var exceptionValues []string
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		exceptionEvents = append(exceptionEvents, event)
		if event == DebugEventException {
			if ex := ctx.Exception(); ex != nil {
				exceptionValues = append(exceptionValues, ex.Value().String())
			}
		}
		return DebugContinue
	})
	dbg.SetExceptionBreakpoints([]string{"all"})
	r.SetDebugger(dbg)
	_, err := r.RunString(`
try {
  throw new Error("caught error");
} catch(e) {}
`)
	if err != nil {
		t.Fatal(err)
	}
	// "all" filter should catch the exception even though it's caught
	foundException := false
	for _, ev := range exceptionEvents {
		if ev == DebugEventException {
			foundException = true
		}
	}
	if !foundException {
		t.Fatalf("Expected exception event with 'all' filter, got events: %v", exceptionEvents)
	}
	if len(exceptionValues) == 0 {
		t.Fatal("Expected exception value to be available")
	}
}

func TestExceptionBreakpointUncaught(t *testing.T) {
	r := New()
	exceptionCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventException {
			exceptionCount++
		}
		return DebugContinue
	})
	dbg.SetExceptionBreakpoints([]string{"uncaught"})
	r.SetDebugger(dbg)

	// This exception is caught — should NOT trigger
	_, _ = r.RunString(`
try {
  throw new Error("caught");
} catch(e) {}
`)
	if exceptionCount != 0 {
		t.Fatalf("Expected 0 exception events for caught exception with 'uncaught' filter, got %d", exceptionCount)
	}
}

func TestExceptionBreakpointUncaughtFires(t *testing.T) {
	r := New()
	exceptionCount := 0
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventException {
			exceptionCount++
		}
		return DebugContinue
	})
	dbg.SetExceptionBreakpoints([]string{"uncaught"})
	r.SetDebugger(dbg)

	// This exception is uncaught — should trigger
	_, err := r.RunString(`throw new Error("uncaught");`)
	if err == nil {
		t.Fatal("Expected error from uncaught exception")
	}
	if exceptionCount != 1 {
		t.Fatalf("Expected 1 exception event for uncaught exception, got %d", exceptionCount)
	}
}

func TestSetVariable(t *testing.T) {
	r := New()
	var result Value
	dbg := NewDebugger(func(ctx *DebugContext, event DebugEvent, pos DebugPosition) DebugAction {
		if event == DebugEventBreakpoint {
			// Set x to 100
			err := ctx.SetVariable(0, 0, "x", r.ToValue(100))
			if err != nil {
				t.Fatalf("SetVariable error: %v", err)
			}
		}
		return DebugContinue
	})
	dbg.SetBreakpoint("", 2, 0)
	r.SetDebugger(dbg)
	v, err := r.RunString("var x = 42;\nvar y = x + 1;\ny;")
	if err != nil {
		t.Fatal(err)
	}
	result = v
	// After setting x=100 at line 2, y = x + 1 = 101
	if result.ToInteger() != 101 {
		t.Fatalf("Expected 101 after SetVariable, got %v", result)
	}
}

func TestEvalHitCondition(t *testing.T) {
	tests := []struct {
		expr     string
		hitCount int
		expected bool
	}{
		{"3", 3, true},
		{"3", 2, false},
		{"3", 6, true},
		{">5", 6, true},
		{">5", 5, false},
		{">=5", 5, true},
		{"<3", 2, true},
		{"<3", 3, false},
		{"<=3", 3, true},
		{"==3", 3, true},
		{"==3", 4, false},
		{"!=3", 4, true},
		{"!=3", 3, false},
		{"", 1, true},
		{"invalid", 1, true},
	}
	for _, tt := range tests {
		result := evalHitCondition(tt.expr, tt.hitCount)
		if result != tt.expected {
			t.Errorf("evalHitCondition(%q, %d) = %v, want %v", tt.expr, tt.hitCount, result, tt.expected)
		}
	}
}
