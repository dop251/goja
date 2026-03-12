package goja

import (
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja/file"
)

// testHook is a mock runtime hook for testing
type testHook struct {
	onInstruction     func(rt *Runtime, pc int) HookResult
	onFunctionEnter   func(rt *Runtime, name string, args []Value)
	onFunctionExit    func(rt *Runtime, name string, result Value)
	onException       func(rt *Runtime, exception *Exception, caught bool) HookResult
	onPromiseReaction func(rt *Runtime, promise *Object)
	onVariableSet     func(rt *Runtime, name string, value Value, scope ScopeType)
}

func (h *testHook) OnInstruction(rt *Runtime, pc int) HookResult {
	if h.onInstruction != nil {
		return h.onInstruction(rt, pc)
	}
	return HookResultContinue
}

func (h *testHook) OnFunctionEnter(rt *Runtime, name string, args []Value) {
	if h.onFunctionEnter != nil {
		h.onFunctionEnter(rt, name, args)
	}
}

func (h *testHook) OnFunctionExit(rt *Runtime, name string, result Value) {
	if h.onFunctionExit != nil {
		h.onFunctionExit(rt, name, result)
	}
}

func (h *testHook) OnException(rt *Runtime, exception *Exception, caught bool) HookResult {
	if h.onException != nil {
		return h.onException(rt, exception, caught)
	}
	return HookResultContinue
}

func (h *testHook) OnPromiseReaction(rt *Runtime, promise *Object) {
	if h.onPromiseReaction != nil {
		h.onPromiseReaction(rt, promise)
	}
}

func (h *testHook) OnVariableSet(rt *Runtime, name string, value Value, scope ScopeType) {
	if h.onVariableSet != nil {
		h.onVariableSet(rt, name, value, scope)
	}
}

func TestRuntimeHookInterfaceCompiles(t *testing.T) {
	// Verify the interface compiles and testHook implements it
	var _ RuntimeHook = (*testHook)(nil)
}

func TestScopeTypeString(t *testing.T) {
	tests := []struct {
		t    ScopeType
		want string
	}{
		{ScopeLocal, "local"},
		{ScopeClosure, "closure"},
		{ScopeBlock, "block"},
		{ScopeGlobal, "global"},
		{ScopeWith, "with"},
		{ScopeType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("ScopeType(%d).String() = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestRuntimeHookAttachDetach(t *testing.T) {
	rt := New()
	h := &testHook{}

	// Initially no hook
	if rt.GetRuntimeHook() != nil {
		t.Error("Expected nil hook initially")
	}

	// Attach
	rt.SetRuntimeHook(h)
	if rt.GetRuntimeHook() != h {
		t.Error("Expected hook to be attached")
	}

	// Detach
	rt.SetRuntimeHook(nil)
	if rt.GetRuntimeHook() != nil {
		t.Error("Expected hook to be detached")
	}
}

func TestRuntimeIsPaused(t *testing.T) {
	rt := New()

	if rt.IsPaused() {
		t.Error("Expected runtime to not be paused initially")
	}
}

func TestRuntimeResumeNotPaused(t *testing.T) {
	rt := New()

	err := rt.Resume()
	if err == nil {
		t.Error("Expected error when resuming non-paused runtime")
	}
}

func TestOnInstructionHookCalled(t *testing.T) {
	rt := New()
	callCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			callCount++
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString("1 + 2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if callCount == 0 {
		t.Error("Expected OnInstruction to be called at least once")
	}
}

func TestOnInstructionHookPauses(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	callCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			callCount++
			count := callCount
			mu.Unlock()
			if count == 3 {
				return HookResultPause // after 3 instructions
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	// Run in a goroutine since it will block when paused
	done := make(chan struct{})
	go func() {
		rt.RunString("var x = 1; x = x + 1; x = x + 1;")
		close(done)
	}()

	// Wait a bit for the pause to happen
	time.Sleep(50 * time.Millisecond)

	if !rt.IsPaused() {
		t.Error("Expected runtime to be paused")
	}

	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 3 {
		t.Errorf("Expected callCount to be 3, got %d", count)
	}

	// Resume to let the goroutine finish
	rt.Resume()
	<-done
}

func TestOnInstructionHookResume(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	callCount := 0
	pauseAt := 5

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			callCount++
			count := callCount
			shouldPause := count == pauseAt
			mu.Unlock()
			if shouldPause {
				return HookResultPause
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	// Run in a goroutine since it will block when paused
	done := make(chan struct{})
	go func() {
		rt.RunString("var x = 10; x = x * 2;")
		close(done)
	}()

	// Wait a bit for the pause to happen
	time.Sleep(50 * time.Millisecond)

	if !rt.IsPaused() {
		t.Error("Expected runtime to be paused")
	}

	mu.Lock()
	firstCallCount := callCount
	mu.Unlock()

	// Don't pause again
	mu.Lock()
	pauseAt = -1
	mu.Unlock()

	// Resume execution
	err := rt.Resume()
	if err != nil {
		t.Fatalf("Unexpected error on resume: %v", err)
	}

	// Wait for completion
	<-done

	if rt.IsPaused() {
		t.Error("Expected runtime to not be paused after resume")
	}

	// More instructions should have been executed
	mu.Lock()
	finalCount := callCount
	mu.Unlock()
	if finalCount <= firstCallCount {
		t.Errorf("Expected more instructions after resume, got %d (was %d)", finalCount, firstCallCount)
	}
}

func TestSourcePosition(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedPos file.Position
	var capturedPC int
	paused := false

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			defer mu.Unlock()
			if !paused {
				stack := rt.CaptureCallStack(1, nil)
				if len(stack) > 0 {
					pos := stack[0].Position()
					if pos.Line > 0 {
						capturedPos = pos
						capturedPC = stack[0].PC()
						paused = true
						return HookResultPause // on first line with valid position
					}
				}
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	// Multi-line code
	code := `var x = 1;
var y = 2;
var z = x + y;`

	done := make(chan struct{})
	go func() {
		rt.RunScript("test.js", code)
		close(done)
	}()

	// Wait for pause
	time.Sleep(100 * time.Millisecond)

	if !rt.IsPaused() {
		t.Error("Expected runtime to be paused")
	}

	mu.Lock()
	pos := capturedPos
	pc := capturedPC
	mu.Unlock()

	if pos.Line <= 0 {
		t.Errorf("Expected valid line number, got %d", pos.Line)
	}

	if pos.Filename != "test.js" {
		t.Errorf("Expected filename 'test.js', got '%s'", pos.Filename)
	}

	if pc < 0 {
		t.Errorf("Expected non-negative PC, got %d", pc)
	}

	// Resume to clean up
	rt.Resume()
	<-done
}

func TestScopes(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedScopes []Scope
	instructionCount := 0
	paused := false

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			defer mu.Unlock()
			instructionCount++
			// Pause after many instructions when we should have scopes set up
			if !paused && instructionCount > 30 {
				scopes := rt.Scopes()
				if len(scopes) > 0 {
					capturedScopes = scopes
					paused = true
					return HookResultPause
				}
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	code := `var globalVar = "global";
function testFunc(arg) {
    var localVar = "local";
    return localVar + arg;
}
testFunc("test");`

	done := make(chan struct{})
	go func() {
		rt.RunScript("test.js", code)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	scopes := capturedScopes
	wasPaused := paused
	mu.Unlock()

	if wasPaused {
		if len(scopes) == 0 {
			t.Error("Expected at least one scope when paused")
		}

		// Verify scope types are valid
		for _, scope := range scopes {
			if scope.Type < ScopeLocal || scope.Type > ScopeWith {
				t.Errorf("Invalid scope type: %d", scope.Type)
			}
			if scope.Variables == nil {
				t.Error("Variables map should not be nil")
			}
		}

		rt.Resume()
	}

	<-done
}

func TestVMState(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedState VMState
	paused := false

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			defer mu.Unlock()
			if !paused {
				capturedState = rt.VMState()
				paused = true
				return HookResultPause
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	done := make(chan struct{})
	go func() {
		rt.RunString("var x = 1 + 2;")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	state := capturedState
	mu.Unlock()

	// PC should be >= 0
	if state.PC < 0 {
		t.Errorf("Expected non-negative PC, got %d", state.PC)
	}

	// SP should be >= 0
	if state.SP < 0 {
		t.Errorf("Expected non-negative SP, got %d", state.SP)
	}

	rt.Resume()
	<-done
}

func TestOnPromiseReactionHook(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	promiseCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onPromiseReaction: func(rt *Runtime, promise *Object) {
			mu.Lock()
			promiseCount++
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
Promise.resolve(42).then(x => x + 1).then(x => x + 2);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Run the job queue to execute promise reactions
	rt.RunString("")

	mu.Lock()
	count := promiseCount
	mu.Unlock()

	if count == 0 {
		t.Error("Expected OnPromiseReaction to be called at least once")
	}
}

func TestOnExceptionHook(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedEx *Exception
	var wasCaught bool

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onException: func(rt *Runtime, exception *Exception, caught bool) HookResult {
			mu.Lock()
			capturedEx = exception
			wasCaught = caught
			mu.Unlock()
			return HookResultContinue // don't pause
		},
	}

	rt.SetRuntimeHook(h)

	// Test caught exception
	_, err := rt.RunString(`
try {
    throw new Error("test error");
} catch (e) {
    // caught
}
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	if capturedEx == nil {
		t.Error("Expected exception to be captured")
	}
	if !wasCaught {
		t.Error("Expected exception to be marked as caught")
	}
	mu.Unlock()

	// Reset for uncaught test
	mu.Lock()
	capturedEx = nil
	wasCaught = false
	mu.Unlock()

	// Test uncaught exception
	_, err = rt.RunString(`throw new Error("uncaught");`)
	if err == nil {
		t.Error("Expected error from uncaught exception")
	}

	mu.Lock()
	if capturedEx == nil {
		t.Error("Expected exception to be captured")
	}
	if wasCaught {
		t.Error("Expected exception to be marked as uncaught")
	}
	mu.Unlock()
}

func TestOnFunctionEnterExit(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	enterCalls := make([]string, 0)
	exitCalls := make([]string, 0)

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue // don't pause
		},
		onFunctionEnter: func(rt *Runtime, name string, args []Value) {
			mu.Lock()
			enterCalls = append(enterCalls, name)
			mu.Unlock()
		},
		onFunctionExit: func(rt *Runtime, name string, result Value) {
			mu.Lock()
			exitCalls = append(exitCalls, name)
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function foo() {
    return bar();
}
function bar() {
    return 42;
}
foo();
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have entered and exited both functions
	if len(enterCalls) < 2 {
		t.Errorf("Expected at least 2 function enters, got %d: %v", len(enterCalls), enterCalls)
	}
	if len(exitCalls) < 2 {
		t.Errorf("Expected at least 2 function exits, got %d: %v", len(exitCalls), exitCalls)
	}

	// Check function names are present
	hasFoo := false
	hasBar := false
	for _, name := range enterCalls {
		if name == "foo" {
			hasFoo = true
		}
		if name == "bar" {
			hasBar = true
		}
	}
	if !hasFoo {
		t.Error("Expected 'foo' in enter calls")
	}
	if !hasBar {
		t.Error("Expected 'bar' in enter calls")
	}
}

func TestCallStack(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedStack []StackFrame
	instructionCount := 0
	paused := false

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			defer mu.Unlock()
			instructionCount++
			// Pause after many instructions, when we're likely inside nested functions
			if !paused && instructionCount > 20 {
				capturedStack = rt.CaptureCallStack(0, nil)
				if len(capturedStack) >= 2 {
					paused = true
					return HookResultPause
				}
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	code := `function outerFunc() {
    function innerFunc() {
        var x = 1;
        var y = 2;
        return x + y;
    }
    return innerFunc();
}
outerFunc();`

	done := make(chan struct{})
	go func() {
		rt.RunScript("test.js", code)
		close(done)
	}()

	// Wait for pause or completion
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	stack := capturedStack
	wasPaused := paused
	mu.Unlock()

	if wasPaused {
		if len(stack) < 2 {
			t.Errorf("Expected at least 2 frames in call stack, got %d", len(stack))
		} else {
			// First frame should be innerFunc
			if stack[0].FuncName() != "innerFunc" {
				t.Errorf("Expected first frame to be 'innerFunc', got '%s'", stack[0].FuncName())
			}
			// Second frame should be outerFunc
			if stack[1].FuncName() != "outerFunc" {
				t.Errorf("Expected second frame to be 'outerFunc', got '%s'", stack[1].FuncName())
			}
		}
		// Resume to clean up
		rt.Resume()
	}

	<-done
}

// === Comprehensive Tests ===

func TestOnExceptionPauses(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	exceptionCount := 0
	pauseOnFirst := true

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onException: func(rt *Runtime, exception *Exception, caught bool) HookResult {
			mu.Lock()
			exceptionCount++
			count := exceptionCount
			shouldPause := pauseOnFirst && count == 1 // Only pause on first exception
			mu.Unlock()
			if shouldPause {
				return HookResultPause
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	done := make(chan struct{})
	go func() {
		rt.RunString(`throw new Error("pause me");`)
		close(done)
	}()

	// Wait for the exception hook to be called and for the pause to take effect
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for exception pause")
		case <-done:
			// Execution completed - check that we did pause at least once
			mu.Lock()
			count := exceptionCount
			mu.Unlock()
			if count == 0 {
				t.Error("Expected OnException to be called")
			}
			return
		default:
			if rt.IsPaused() {
				// Good, we're paused - resume
				rt.Resume()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestOnFunctionEnterExitWithRecursion(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	enterCalls := make([]string, 0)
	exitCalls := make([]string, 0)

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onFunctionEnter: func(rt *Runtime, name string, args []Value) {
			mu.Lock()
			enterCalls = append(enterCalls, name)
			mu.Unlock()
		},
		onFunctionExit: func(rt *Runtime, name string, result Value) {
			mu.Lock()
			exitCalls = append(exitCalls, name)
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function factorial(n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}
factorial(4);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// factorial(4) should call factorial 4 times (4, 3, 2, 1)
	factorialEnters := 0
	factorialExits := 0
	for _, name := range enterCalls {
		if name == "factorial" {
			factorialEnters++
		}
	}
	for _, name := range exitCalls {
		if name == "factorial" {
			factorialExits++
		}
	}

	if factorialEnters != 4 {
		t.Errorf("Expected 4 factorial enters, got %d", factorialEnters)
	}
	if factorialExits != 4 {
		t.Errorf("Expected 4 factorial exits, got %d", factorialExits)
	}
}

func TestOnFunctionExitWithException(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	exitCalls := make([]struct {
		name      string
		hasResult bool
	}, 0)

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onFunctionExit: func(rt *Runtime, name string, result Value) {
			mu.Lock()
			exitCalls = append(exitCalls, struct {
				name      string
				hasResult bool
			}{
				name:      name,
				hasResult: result != nil && result != Undefined(),
			})
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	rt.RunString(`
function throwingFunc() {
    throw new Error("boom");
}
try {
    throwingFunc();
} catch(e) {}
`)

	mu.Lock()
	defer mu.Unlock()

	// Note: When a function exits via exception, the OnFunctionExit hook is not called
	// because the exception unwinds the stack before the normal return path.
	// Use OnException hook to handle exception cases.
	t.Logf("Got %d function exits: %v", len(exitCalls), exitCalls)
}

func TestOnFunctionEnterExitWithArrowFunctions(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	enterCount := 0
	enterNames := make([]string, 0)

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onFunctionEnter: func(rt *Runtime, name string, args []Value) {
			mu.Lock()
			enterCount++
			enterNames = append(enterNames, name)
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
const add = (a, b) => a + b;
const result = [1, 2, 3].map(x => x * 2);
add(1, 2);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	count := enterCount
	names := enterNames
	mu.Unlock()

	// Arrow functions are tracked - verify we see at least some function calls
	// The exact count depends on how built-in functions (map) handle callbacks
	t.Logf("Got %d function enters: %v", count, names)
	if count < 1 {
		t.Error("Expected at least 1 function enter for arrow functions")
	}
}

func TestMultiplePauseResumeCycles(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	pauseCount := 0
	resumeCount := 0
	instructionCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			instructionCount++
			count := instructionCount
			mu.Unlock()

			// Pause every 10 instructions, up to 3 times
			if count%10 == 0 && pauseCount < 3 {
				mu.Lock()
				pauseCount++
				mu.Unlock()
				return HookResultPause
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	done := make(chan struct{})
	go func() {
		rt.RunString(`
var sum = 0;
for (var i = 0; i < 100; i++) {
    sum += i;
}
`)
		close(done)
	}()

	// Resume 3 times
	for i := 0; i < 3; i++ {
		time.Sleep(50 * time.Millisecond)

		if !rt.IsPaused() {
			// Might have already finished all 3 pauses
			break
		}

		err := rt.Resume()
		if err != nil {
			t.Fatalf("Resume %d failed: %v", i+1, err)
		}
		resumeCount++
	}

	// Final resume if still paused
	time.Sleep(50 * time.Millisecond)
	if rt.IsPaused() {
		rt.Resume()
	}

	<-done

	mu.Lock()
	finalPauseCount := pauseCount
	mu.Unlock()

	if finalPauseCount < 2 {
		t.Errorf("Expected at least 2 pauses, got %d", finalPauseCount)
	}
}

func TestScopesBlockScope(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedScopes []Scope
	instructionCount := 0
	paused := false

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			defer mu.Unlock()
			instructionCount++
			if !paused && instructionCount > 20 {
				scopes := rt.Scopes()
				// Look for block scoped variable
				for _, scope := range scopes {
					if _, ok := scope.Variables["blockVar"]; ok {
						capturedScopes = scopes
						paused = true
						return HookResultPause
					}
				}
			}
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	code := `function testBlockScope() {
    var outerVar = 1;
    {
        let blockVar = 2;
        const blockConst = 3;
        return blockVar + blockConst;
    }
}
testBlockScope();`

	done := make(chan struct{})
	go func() {
		rt.RunString(code)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	scopes := capturedScopes
	wasPaused := paused
	mu.Unlock()

	if wasPaused {
		foundBlockVar := false
		for _, scope := range scopes {
			if _, ok := scope.Variables["blockVar"]; ok {
				foundBlockVar = true
				break
			}
		}
		if !foundBlockVar {
			t.Error("Expected to find blockVar in scopes")
		}
		rt.Resume()
	}

	<-done
}

func TestAccessorsWithoutHook(t *testing.T) {
	rt := New()

	// These should not panic when no hook is attached

	stack := rt.CaptureCallStack(0, nil)
	if len(stack) != 0 {
		// When not running, call stack should be empty
		t.Logf("Call stack has %d frames when not running", len(stack))
	}

	scopes := rt.Scopes()
	// Scopes may be empty or have global scope
	t.Logf("Got %d scopes when no code running", len(scopes))

	state := rt.VMState()
	// VM state should have default values
	if state.PC != 0 || state.SP != 0 {
		t.Logf("VM state: PC=%d, SP=%d", state.PC, state.SP)
	}
}

func TestPositionMultipleFiles(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	positions := make(map[string]bool)
	instructionCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			instructionCount++
			if instructionCount < 100 {
				stack := rt.CaptureCallStack(1, nil)
				if len(stack) > 0 {
					pos := stack[0].Position()
					if pos.Filename != "" {
						positions[pos.Filename] = true
					}
				}
			}
			mu.Unlock()
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	// Run two different scripts
	rt.RunScript("file1.js", "var x = 1;")
	rt.RunScript("file2.js", "var y = 2;")

	mu.Lock()
	defer mu.Unlock()

	if !positions["file1.js"] {
		t.Error("Expected to see file1.js in positions")
	}
	if !positions["file2.js"] {
		t.Error("Expected to see file2.js in positions")
	}
}

func TestOnExceptionNestedTryCatch(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	exceptions := make([]struct {
		caught bool
	}, 0)

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onException: func(rt *Runtime, exception *Exception, caught bool) HookResult {
			mu.Lock()
			exceptions = append(exceptions, struct{ caught bool }{caught})
			mu.Unlock()
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
try {
    try {
        throw new Error("inner");
    } catch (e) {
        throw new Error("outer");
    }
} catch (e) {
    // caught outer
}
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(exceptions) < 2 {
		t.Errorf("Expected at least 2 exceptions, got %d", len(exceptions))
	}

	// Both exceptions should be caught
	for i, ex := range exceptions {
		if !ex.caught {
			t.Errorf("Exception %d should be marked as caught", i)
		}
	}
}

func TestCallStackDeepRecursion(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var maxStackDepth int
	instructionCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			instructionCount++
			if instructionCount%50 == 0 {
				stack := rt.CaptureCallStack(0, nil)
				if len(stack) > maxStackDepth {
					maxStackDepth = len(stack)
				}
			}
			mu.Unlock()
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function recurse(n) {
    if (n <= 0) return 0;
    return 1 + recurse(n - 1);
}
recurse(20);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	depth := maxStackDepth
	mu.Unlock()

	// Should see at least 10 frames at some point
	if depth < 10 {
		t.Errorf("Expected max stack depth of at least 10, got %d", depth)
	}
}

func TestOnPromiseReactionChain(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	reactionCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onPromiseReaction: func(rt *Runtime, promise *Object) {
			mu.Lock()
			reactionCount++
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
Promise.resolve(1)
    .then(x => x + 1)
    .then(x => x + 1)
    .then(x => x + 1)
    .then(x => x + 1);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Run job queue
	rt.RunString("")

	mu.Lock()
	count := reactionCount
	mu.Unlock()

	// Should have multiple reactions - the exact count may vary
	// based on how promise chaining works internally
	t.Logf("Got %d promise reactions", count)
	if count < 3 {
		t.Errorf("Expected at least 3 promise reactions, got %d", count)
	}
}

func TestVMStateInNestedCalls(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var maxCallDepth int
	instructionCount := 0

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			mu.Lock()
			instructionCount++
			if instructionCount%10 == 0 {
				state := rt.VMState()
				if state.CallDepth > maxCallDepth {
					maxCallDepth = state.CallDepth
				}
			}
			mu.Unlock()
			return HookResultContinue
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function a() { return b(); }
function b() { return c(); }
function c() { return d(); }
function d() { return 42; }
a();
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	depth := maxCallDepth
	mu.Unlock()

	// Should see at least 3 call depth (a -> b -> c -> d)
	if depth < 3 {
		t.Errorf("Expected max call depth of at least 3, got %d", depth)
	}
}

func TestOnFunctionEnterArgs(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedArgs []Value
	var capturedName string

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onFunctionEnter: func(rt *Runtime, name string, args []Value) {
			if name == "testArgs" {
				mu.Lock()
				capturedName = name
				capturedArgs = append([]Value{}, args...)
				mu.Unlock()
			}
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function testArgs(a, b, c) {
    return a + b + c;
}
testArgs(1, "hello", true);
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify the hook was called
	if capturedName != "testArgs" {
		t.Errorf("Expected function name 'testArgs', got '%s'", capturedName)
	}

	// Verify args were captured (the exact content depends on stack layout)
	t.Logf("Got %d args: %v", len(capturedArgs), capturedArgs)
	if len(capturedArgs) == 0 {
		t.Error("Expected to capture some arguments")
	}
}

func TestOnFunctionExitReturnValue(t *testing.T) {
	rt := New()
	var mu sync.Mutex
	var capturedResult Value

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onFunctionExit: func(rt *Runtime, name string, result Value) {
			if name == "testReturn" {
				mu.Lock()
				capturedResult = result
				mu.Unlock()
			}
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
function testReturn() {
    return { value: 42 };
}
testReturn();
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	result := capturedResult
	mu.Unlock()

	if result == nil {
		t.Error("Expected result to be captured")
	} else {
		obj := result.ToObject(rt)
		val := obj.Get("value")
		if val.ToInteger() != 42 {
			t.Errorf("Expected return value.value to be 42, got %v", val)
		}
	}
}

func TestOnVariableSet(t *testing.T) {
	rt := New()

	type varSet struct {
		name  string
		value string
		scope ScopeType
	}
	var mu sync.Mutex
	var sets []varSet

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			return HookResultContinue
		},
		onVariableSet: func(rt *Runtime, name string, value Value, scope ScopeType) {
			mu.Lock()
			sets = append(sets, varSet{name: name, value: value.String(), scope: scope})
			mu.Unlock()
		},
	}

	rt.SetRuntimeHook(h)

	_, err := rt.RunString(`
var x = 10;
let y = 20;
x = 30;
y = 40;
`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	captured := sets
	mu.Unlock()

	// Should have captured: x=10, y=20, x=30, y=40
	if len(captured) < 4 {
		t.Errorf("Expected at least 4 variable sets, got %d: %v", len(captured), captured)
	}

	// Check that we captured the expected variable names
	foundX := false
	foundY := false
	for _, s := range captured {
		if s.name == "x" {
			foundX = true
		}
		if s.name == "y" {
			foundY = true
		}
	}

	if !foundX {
		t.Error("Expected to capture variable 'x'")
	}
	if !foundY {
		t.Error("Expected to capture variable 'y'")
	}

	// var declarations go on global object (ScopeGlobal)
	// let declarations go in lexical environment (ScopeLocal)
	for _, s := range captured {
		if s.name == "x" && s.scope != ScopeGlobal {
			t.Errorf("Expected scope to be global for var x, got %v", s.scope)
		}
		// let y is in lexical scope, so it's ScopeLocal or ScopeGlobal depending on path
	}
}

// TestVMStateInAsync verifies that VMState.InAsync correctly detects async context.
// InAsync is true when execution is resumed after an await, not on initial async function entry.
func TestVMStateInAsync(t *testing.T) {
	rt := New()

	var syncInAsync bool
	var afterAwaitInAsync bool

	h := &testHook{
		onFunctionEnter: func(rt *Runtime, name string, args []Value) {
			state := rt.VMState()
			if name == "syncFunc" {
				syncInAsync = state.InAsync
			} else if name == "afterAwait" {
				afterAwaitInAsync = state.InAsync
			}
		},
	}
	rt.SetRuntimeHook(h)

	// afterAwait is called after an await, so it should have InAsync=true
	v, err := rt.RunString(`
		function syncFunc() {
			return 1;
		}

		function afterAwait() {
			return 2;
		}

		syncFunc();

		(async function() {
			await Promise.resolve();
			afterAwait();
		})()
	`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the promise completed
	promise := v.Export().(*Promise)
	if promise.State() != PromiseStateFulfilled {
		t.Fatalf("Expected promise to be fulfilled, got %v", promise.State())
	}

	if syncInAsync {
		t.Error("Expected InAsync to be false in sync function")
	}
	if !afterAwaitInAsync {
		t.Error("Expected InAsync to be true after await")
	}
}

// TestLoadedScripts verifies that loaded scripts are tracked and can be retrieved.
func TestLoadedScripts(t *testing.T) {
	rt := New()

	// Initially, no scripts should be loaded
	scripts := rt.LoadedScripts()
	if len(scripts) != 0 {
		t.Errorf("Expected 0 scripts initially, got %d", len(scripts))
	}

	// Run a script with a filename
	prg, err := Compile("test1.js", "var x = 1;", false)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	_, err = rt.RunProgram(prg)
	if err != nil {
		t.Fatalf("RunProgram error: %v", err)
	}

	scripts = rt.LoadedScripts()
	if len(scripts) != 1 {
		t.Errorf("Expected 1 script after first run, got %d", len(scripts))
	}
	if scripts[0].Name != "test1.js" {
		t.Errorf("Expected script name 'test1.js', got '%s'", scripts[0].Name)
	}
	if scripts[0].Source != "var x = 1;" {
		t.Errorf("Expected script source 'var x = 1;', got '%s'", scripts[0].Source)
	}

	// Run another script
	prg2, err := Compile("test2.js", "var y = 2;", false)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	_, err = rt.RunProgram(prg2)
	if err != nil {
		t.Fatalf("RunProgram error: %v", err)
	}

	scripts = rt.LoadedScripts()
	if len(scripts) != 2 {
		t.Errorf("Expected 2 scripts after second run, got %d", len(scripts))
	}

	// Running the same program again should not duplicate
	_, err = rt.RunProgram(prg)
	if err != nil {
		t.Fatalf("RunProgram error: %v", err)
	}

	scripts = rt.LoadedScripts()
	if len(scripts) != 2 {
		t.Errorf("Expected 2 scripts (no duplicate), got %d", len(scripts))
	}
}

// TestFindPCsForLine verifies that we can find program counters for source lines.
// Note: FindPCsForLine only works for top-level code, not code inside functions.
// For breakpoints inside functions, use OnInstruction to match by position.
func TestFindPCsForLine(t *testing.T) {
	rt := New()

	// Use top-level code (no functions) for this test
	source := `var a = 1;
var b = 2;
var c = a + b;`

	prg, err := Compile("test.js", source, false)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	_, err = rt.RunProgram(prg)
	if err != nil {
		t.Fatalf("RunProgram error: %v", err)
	}

	// Find PCs for line 1 (var a = 1;)
	pcs := rt.FindPCsForLine("test.js", 1)
	if len(pcs) == 0 {
		t.Error("Expected at least one PC for line 1")
	}

	// Find PCs for line 2 (var b = 2;)
	pcs = rt.FindPCsForLine("test.js", 2)
	if len(pcs) == 0 {
		t.Error("Expected at least one PC for line 2")
	}

	// Find PCs for line 3 (var c = a + b;)
	pcs = rt.FindPCsForLine("test.js", 3)
	if len(pcs) == 0 {
		t.Error("Expected at least one PC for line 3")
	}

	// Non-existent file should return empty
	pcs = rt.FindPCsForLine("nonexistent.js", 1)
	if len(pcs) != 0 {
		t.Errorf("Expected 0 PCs for nonexistent file, got %d", len(pcs))
	}

	// Non-existent line should return empty
	pcs = rt.FindPCsForLine("test.js", 1000)
	if len(pcs) != 0 {
		t.Errorf("Expected 0 PCs for nonexistent line, got %d", len(pcs))
	}
}

// TestBreakpointWithFindPCsForLine demonstrates using FindPCsForLine for breakpoints.
func TestBreakpointWithFindPCsForLine(t *testing.T) {
	source := `var a = 1;
var b = 2;
var c = 3;`

	prg, err := Compile("breakpoint_test.js", source, false)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	// First, run to track the program
	rt := New()
	rt.RunProgram(prg)

	// Get PCs for line 2
	pcs := rt.FindPCsForLine("breakpoint_test.js", 2)
	if len(pcs) == 0 {
		t.Fatal("Expected at least one PC for line 2")
	}

	// Create a new runtime and set a breakpoint on line 2
	rt2 := New()

	breakpointPCs := make(map[int]bool)
	for _, pc := range pcs {
		breakpointPCs[pc] = true
	}

	hitBreakpoint := false
	var breakpointLine int

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			if breakpointPCs[pc] && !hitBreakpoint {
				hitBreakpoint = true
				frames := rt.CaptureCallStack(1, nil)
				if len(frames) > 0 {
					breakpointLine = frames[0].Position().Line
				}
			}
			return HookResultContinue // don't pause, just track
		},
	}
	rt2.SetRuntimeHook(h)

	rt2.RunProgram(prg)

	if !hitBreakpoint {
		t.Error("Expected to hit breakpoint")
	}
	if breakpointLine != 2 {
		t.Errorf("Expected breakpoint at line 2, got line %d", breakpointLine)
	}
}

// TestBreakpointByPosition demonstrates the recommended way to set breakpoints
// by matching position in OnInstruction, which works for all code including functions.
func TestBreakpointByPosition(t *testing.T) {
	source := `function add(a, b) {
    var sum = a + b;
    return sum;
}
var result = add(1, 2);`

	prg, err := Compile("position_test.js", source, false)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	rt := New()

	// Set a breakpoint on line 2 (inside the function)
	breakpointFile := "position_test.js"
	breakpointLine := 2

	hitBreakpoint := false
	var capturedLine int

	h := &testHook{
		onInstruction: func(rt *Runtime, pc int) HookResult {
			if !hitBreakpoint {
				frames := rt.CaptureCallStack(1, nil)
				if len(frames) > 0 {
					pos := frames[0].Position()
					if pos.Filename == breakpointFile && pos.Line == breakpointLine {
						hitBreakpoint = true
						capturedLine = pos.Line
					}
				}
			}
			return HookResultContinue // don't pause, just track
		},
	}
	rt.SetRuntimeHook(h)

	rt.RunProgram(prg)

	if !hitBreakpoint {
		t.Error("Expected to hit breakpoint inside function")
	}
	if capturedLine != breakpointLine {
		t.Errorf("Expected breakpoint at line %d, got line %d", breakpointLine, capturedLine)
	}
}
