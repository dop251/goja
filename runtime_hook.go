package goja

// HookResult indicates what action the runtime should take after a hook returns.
type HookResult int

const (
	// HookResultContinue tells the runtime to continue execution normally.
	HookResultContinue HookResult = iota

	// HookResultPause tells the runtime to pause execution.
	// Execution will resume when Runtime.Resume() is called.
	HookResultPause
)

// RuntimeHook is the interface for runtime instrumentation.
// goja calls these methods at various execution points.
// Can be used to build debuggers, profilers, tracers, coverage tools, etc.
//
// For convenience, embed BaseRuntimeHook to get no-op implementations
// of all methods, then override only the ones you need.
type RuntimeHook interface {
	// OnInstruction is called before each VM instruction.
	// pc is the program counter.
	OnInstruction(rt *Runtime, pc int) HookResult

	// OnFunctionEnter is called when entering a JS function.
	OnFunctionEnter(rt *Runtime, name string, args []Value)

	// OnFunctionExit is called when exiting a JS function.
	// result is the return value.
	OnFunctionExit(rt *Runtime, name string, result Value)

	// OnException is called when an exception is thrown.
	// caught is true if the exception will be caught by a try/catch.
	OnException(rt *Runtime, exception *Exception, caught bool) HookResult

	// OnPromiseReaction is called when a promise reaction is about to be enqueued.
	OnPromiseReaction(rt *Runtime, promise *Object)

	// OnVariableSet is called when a variable is assigned a value.
	// name is the variable name, value is the new value being assigned.
	// scope indicates where the variable lives (global, local, closure, etc.)
	OnVariableSet(rt *Runtime, name string, value Value, scope ScopeType)
}

// BaseRuntimeHook provides no-op implementations of all RuntimeHook methods.
// Embed this struct and override only the methods you need.
//
// Example:
//
//	type MyHook struct {
//	    goja.BaseRuntimeHook
//	}
//
//	func (h *MyHook) OnInstruction(rt *goja.Runtime, pc int) goja.HookResult {
//	    // your implementation
//	    return goja.HookResultContinue
//	}
type BaseRuntimeHook struct{}

func (BaseRuntimeHook) OnInstruction(rt *Runtime, pc int) HookResult {
	return HookResultContinue
}

func (BaseRuntimeHook) OnFunctionEnter(rt *Runtime, name string, args []Value) {}

func (BaseRuntimeHook) OnFunctionExit(rt *Runtime, name string, result Value) {}

func (BaseRuntimeHook) OnException(rt *Runtime, exception *Exception, caught bool) HookResult {
	return HookResultContinue
}

func (BaseRuntimeHook) OnPromiseReaction(rt *Runtime, promise *Object) {}

func (BaseRuntimeHook) OnVariableSet(rt *Runtime, name string, value Value, scope ScopeType) {}
