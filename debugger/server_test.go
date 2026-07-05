package debugger

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dop251/goja"
	dap "github.com/google/go-dap"
)

// testClient wraps a DAP transport for testing.
type testClient struct {
	reader *bufio.Reader
	writer io.Writer
	seq    int
	t      *testing.T
}

func newTestClient(t *testing.T, reader io.Reader, writer io.Writer) *testClient {
	return &testClient{
		reader: bufio.NewReader(reader),
		writer: writer,
		t:      t,
	}
}

func (c *testClient) send(msg dap.Message) {
	c.seq++
	setSeq(msg, c.seq)
	if err := dap.WriteProtocolMessage(c.writer, msg); err != nil {
		c.t.Fatalf("Failed to send DAP message: %v", err)
	}
}

func (c *testClient) read() dap.Message {
	msg, err := dap.ReadProtocolMessage(c.reader)
	if err != nil {
		c.t.Fatalf("Failed to read DAP message: %v", err)
	}
	return msg
}

func (c *testClient) expectResponse(command string) dap.Message {
	msg := c.read()
	if resp, ok := msg.(dap.ResponseMessage); ok {
		r := resp.GetResponse()
		if r.Command != command {
			c.t.Fatalf("Expected %s response, got %s", command, r.Command)
		}
		if !r.Success {
			c.t.Fatalf("Expected success response for %s, got failure: %s", command, r.Message)
		}
		return msg
	}
	c.t.Fatalf("Expected response message, got %T", msg)
	return nil
}

func (c *testClient) expectEvent(event string) dap.Message {
	msg := c.read()
	if evt, ok := msg.(dap.EventMessage); ok {
		e := evt.GetEvent()
		if e.Event != event {
			c.t.Fatalf("Expected %s event, got %s", event, e.Event)
		}
		return msg
	}
	c.t.Fatalf("Expected event message, got %T", msg)
	return nil
}

func (c *testClient) initialize() {
	c.send(&dap.InitializeRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "initialize",
		},
		Arguments: dap.InitializeRequestArguments{
			AdapterID: "goja-test",
		},
	})
	c.expectResponse("initialize")
	c.expectEvent("initialized")
}

func (c *testClient) launch(stopOnEntry bool) {
	args := "{}"
	if stopOnEntry {
		args = `{"stopOnEntry": true}`
	}
	c.send(&dap.LaunchRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "launch",
		},
		Arguments: []byte(args),
	})
	c.expectResponse("launch")
}

func (c *testClient) setBreakpoints(path string, lines ...int) *dap.SetBreakpointsResponse {
	bps := make([]dap.SourceBreakpoint, len(lines))
	for i, line := range lines {
		bps[i] = dap.SourceBreakpoint{Line: line}
	}
	c.send(&dap.SetBreakpointsRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "setBreakpoints",
		},
		Arguments: dap.SetBreakpointsArguments{
			Source:      dap.Source{Path: path},
			Breakpoints: bps,
		},
	})
	resp := c.expectResponse("setBreakpoints")
	return resp.(*dap.SetBreakpointsResponse)
}

func (c *testClient) configurationDone() {
	c.send(&dap.ConfigurationDoneRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "configurationDone",
		},
	})
	c.expectResponse("configurationDone")
}

func (c *testClient) continueExec() {
	c.send(&dap.ContinueRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "continue",
		},
		Arguments: dap.ContinueArguments{ThreadId: 1},
	})
	c.expectResponse("continue")
}

func (c *testClient) next() {
	c.send(&dap.NextRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "next",
		},
		Arguments: dap.NextArguments{ThreadId: 1},
	})
	c.expectResponse("next")
}

func (c *testClient) stepIn() {
	c.send(&dap.StepInRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "stepIn",
		},
		Arguments: dap.StepInArguments{ThreadId: 1},
	})
	c.expectResponse("stepIn")
}

func (c *testClient) stepOut() {
	c.send(&dap.StepOutRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "stepOut",
		},
		Arguments: dap.StepOutArguments{ThreadId: 1},
	})
	c.expectResponse("stepOut")
}

func (c *testClient) stackTrace() *dap.StackTraceResponse {
	c.send(&dap.StackTraceRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "stackTrace",
		},
		Arguments: dap.StackTraceArguments{ThreadId: 1},
	})
	resp := c.expectResponse("stackTrace")
	return resp.(*dap.StackTraceResponse)
}

func (c *testClient) scopes(frameId int) *dap.ScopesResponse {
	c.send(&dap.ScopesRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "scopes",
		},
		Arguments: dap.ScopesArguments{FrameId: frameId},
	})
	resp := c.expectResponse("scopes")
	return resp.(*dap.ScopesResponse)
}

func (c *testClient) variables(ref int) *dap.VariablesResponse {
	c.send(&dap.VariablesRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "variables",
		},
		Arguments: dap.VariablesArguments{VariablesReference: ref},
	})
	resp := c.expectResponse("variables")
	return resp.(*dap.VariablesResponse)
}

func (c *testClient) evaluate(expr string, frameId int) *dap.EvaluateResponse {
	c.send(&dap.EvaluateRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "evaluate",
		},
		Arguments: dap.EvaluateArguments{
			Expression: expr,
			FrameId:    frameId,
		},
	})
	resp := c.expectResponse("evaluate")
	return resp.(*dap.EvaluateResponse)
}

func (c *testClient) setBreakpointsWithOpts(path string, sbps ...dap.SourceBreakpoint) *dap.SetBreakpointsResponse {
	c.send(&dap.SetBreakpointsRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "setBreakpoints",
		},
		Arguments: dap.SetBreakpointsArguments{
			Source:      dap.Source{Path: path},
			Breakpoints: sbps,
		},
	})
	resp := c.expectResponse("setBreakpoints")
	return resp.(*dap.SetBreakpointsResponse)
}

func (c *testClient) setExceptionBreakpoints(filters ...string) {
	c.send(&dap.SetExceptionBreakpointsRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "setExceptionBreakpoints",
		},
		Arguments: dap.SetExceptionBreakpointsArguments{
			Filters: filters,
		},
	})
	c.expectResponse("setExceptionBreakpoints")
}

func (c *testClient) setVariable(ref int, name, value string) *dap.SetVariableResponse {
	c.send(&dap.SetVariableRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "setVariable",
		},
		Arguments: dap.SetVariableArguments{
			VariablesReference: ref,
			Name:               name,
			Value:              value,
		},
	})
	resp := c.expectResponse("setVariable")
	return resp.(*dap.SetVariableResponse)
}

func (c *testClient) disconnect() {
	c.send(&dap.DisconnectRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "disconnect",
		},
	})
	c.expectResponse("disconnect")
}

func (c *testClient) threads() *dap.ThreadsResponse {
	c.send(&dap.ThreadsRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "threads",
		},
	})
	resp := c.expectResponse("threads")
	return resp.(*dap.ThreadsResponse)
}

// setupServer creates a server and client connected via pipes.
func setupServer(t *testing.T, script string) (*Server, *testClient) {
	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()

	r := goja.New()
	srv := NewServer(r, serverReader, serverWriter, func() error {
		_, err := r.RunString(script)
		return err
	})

	client := newTestClient(t, clientReader, clientWriter)
	return srv, client
}

func TestFullSession(t *testing.T) {
	srv, client := setupServer(t, "var x = 1;\nvar y = 2;\nvar z = 3;")

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run()
	}()

	// Initialize
	client.initialize()

	// Launch
	client.launch(false)

	// Set breakpoint on line 2
	bpResp := client.setBreakpoints("", 2)
	if len(bpResp.Body.Breakpoints) != 1 {
		t.Fatalf("Expected 1 breakpoint, got %d", len(bpResp.Body.Breakpoints))
	}
	if !bpResp.Body.Breakpoints[0].Verified {
		t.Fatal("Expected breakpoint to be verified")
	}

	// Configuration done — starts VM
	client.configurationDone()

	// Should hit breakpoint on line 2
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("Expected reason 'breakpoint', got '%s'", stopped.Body.Reason)
	}

	// Threads
	threadsResp := client.threads()
	if len(threadsResp.Body.Threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(threadsResp.Body.Threads))
	}
	if threadsResp.Body.Threads[0].Name != "main" {
		t.Fatalf("Expected thread name 'main', got '%s'", threadsResp.Body.Threads[0].Name)
	}

	// Stack trace
	stResp := client.stackTrace()
	if len(stResp.Body.StackFrames) == 0 {
		t.Fatal("Expected at least 1 stack frame")
	}
	if stResp.Body.StackFrames[0].Line != 2 {
		t.Fatalf("Expected frame at line 2, got line %d", stResp.Body.StackFrames[0].Line)
	}

	// Continue
	client.continueExec()

	// Should get terminated event
	client.expectEvent("terminated")

	// Disconnect
	client.disconnect()

	// Server should exit
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not exit in time")
	}
}

func TestStepping(t *testing.T) {
	srv, client := setupServer(t, `var a = 1;
var b = 2;
var c = 3;
var d = 4;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.setBreakpoints("", 1)
	client.configurationDone()

	// Hit breakpoint at line 1
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}

	// Step over → should stop at line 2
	client.next()
	stopped = client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "step" {
		t.Fatalf("Expected 'step', got '%s'", stopped.Body.Reason)
	}
	st := client.stackTrace()
	if st.Body.StackFrames[0].Line != 2 {
		t.Fatalf("Expected line 2 after step over, got %d", st.Body.StackFrames[0].Line)
	}

	// Step over again → line 3
	client.next()
	client.expectEvent("stopped")
	st = client.stackTrace()
	if st.Body.StackFrames[0].Line != 3 {
		t.Fatalf("Expected line 3, got %d", st.Body.StackFrames[0].Line)
	}

	// Continue to end
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestStepInOut(t *testing.T) {
	srv, client := setupServer(t, `function foo() {
  var inner = 42;
  return inner;
}
var result = foo();`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.setBreakpoints("", 5) // Break at call site
	client.configurationDone()

	// Hit breakpoint at line 5
	client.expectEvent("stopped")

	// Step in → should enter foo()
	client.stepIn()
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "step" {
		t.Fatalf("Expected 'step', got '%s'", stopped.Body.Reason)
	}
	st := client.stackTrace()
	if len(st.Body.StackFrames) < 2 {
		t.Fatalf("Expected at least 2 frames after step-in, got %d", len(st.Body.StackFrames))
	}

	// Step out → should return to call site
	client.stepOut()
	client.expectEvent("stopped")

	// Continue to end
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestVariableInspection(t *testing.T) {
	srv, client := setupServer(t, `var x = 42;
var y = "hello";
var obj = {a: 1, b: 2};
debugger;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Should hit debugger statement
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" { // debugger stmt maps to "breakpoint"
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}

	// Get stack trace
	st := client.stackTrace()
	if len(st.Body.StackFrames) == 0 {
		t.Fatal("Expected at least 1 stack frame")
	}
	frameId := st.Body.StackFrames[0].Id

	// Get scopes
	scopesResp := client.scopes(frameId)
	if len(scopesResp.Body.Scopes) == 0 {
		t.Fatal("Expected at least 1 scope")
	}

	// Get variables from the first scope that has variables
	var foundX, foundY, foundObj bool
	for _, scope := range scopesResp.Body.Scopes {
		varsResp := client.variables(scope.VariablesReference)
		for _, v := range varsResp.Body.Variables {
			switch v.Name {
			case "x":
				foundX = true
				if v.Value != "42" {
					t.Fatalf("Expected x=42, got x=%s", v.Value)
				}
			case "y":
				foundY = true
				if v.Value != "hello" {
					t.Fatalf("Expected y=hello, got y=%s", v.Value)
				}
			case "obj":
				foundObj = true
				if v.VariablesReference == 0 {
					t.Fatal("Expected obj to be expandable (variablesReference > 0)")
				}
				// Expand the object
				objVars := client.variables(v.VariablesReference)
				var foundA, foundB bool
				for _, ov := range objVars.Body.Variables {
					if ov.Name == "a" && ov.Value == "1" {
						foundA = true
					}
					if ov.Name == "b" && ov.Value == "2" {
						foundB = true
					}
				}
				if !foundA || !foundB {
					t.Fatalf("Expected obj to have a=1 and b=2, got %+v", objVars.Body.Variables)
				}
			}
		}
	}

	if !foundX || !foundY || !foundObj {
		t.Fatalf("Missing variables: x=%v y=%v obj=%v", foundX, foundY, foundObj)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestEvaluate(t *testing.T) {
	srv, client := setupServer(t, `var x = 10;
var y = 20;
debugger;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Hit debugger statement
	client.expectEvent("stopped")

	st := client.stackTrace()
	frameId := st.Body.StackFrames[0].Id

	// Evaluate simple expression
	evalResp := client.evaluate("x + y", frameId)
	if evalResp.Body.Result != "30" {
		t.Fatalf("Expected '30', got '%s'", evalResp.Body.Result)
	}

	// Evaluate expression with side effects
	evalResp = client.evaluate("x = 100", frameId)
	if evalResp.Body.Result != "100" {
		t.Fatalf("Expected '100', got '%s'", evalResp.Body.Result)
	}

	// Verify side effect persisted
	evalResp = client.evaluate("x", frameId)
	if evalResp.Body.Result != "100" {
		t.Fatalf("Expected x to be 100 after mutation, got '%s'", evalResp.Body.Result)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestPause(t *testing.T) {
	// Use a script with a loop so there's time to pause
	srv, client := setupServer(t, `var i = 0;
while (i < 1000000) {
  i++;
}`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Request pause while running
	client.send(&dap.PauseRequest{
		Request: dap.Request{
			ProtocolMessage: dap.ProtocolMessage{Type: "request"},
			Command:         "pause",
		},
		Arguments: dap.PauseArguments{ThreadId: 1},
	})
	client.expectResponse("pause")

	// Should get stopped event with reason "pause"
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "pause" {
		t.Fatalf("Expected reason 'pause', got '%s'", stopped.Body.Reason)
	}

	// Continue to finish
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDebuggerStatement(t *testing.T) {
	srv, client := setupServer(t, `var x = 1;
debugger;
var y = 2;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Should hit debugger statement
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" { // debugger stmt maps to "breakpoint"
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestStopOnEntry(t *testing.T) {
	srv, client := setupServer(t, `var x = 1;
var y = 2;`)

	go srv.Run()

	client.initialize()
	client.launch(true) // stopOnEntry = true
	client.configurationDone()

	// Should pause at entry
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "pause" {
		t.Fatalf("Expected reason 'pause' for stop-on-entry, got '%s'", stopped.Body.Reason)
	}

	// Verify we're at line 1
	st := client.stackTrace()
	if len(st.Body.StackFrames) > 0 && st.Body.StackFrames[0].Line != 1 {
		t.Fatalf("Expected to stop at line 1, got line %d", st.Body.StackFrames[0].Line)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDisconnectWhilePaused(t *testing.T) {
	srv, client := setupServer(t, `debugger;
var x = 1;`)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run()
	}()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Hit debugger statement
	client.expectEvent("stopped")

	// Disconnect while paused
	client.disconnect()

	// Server should exit cleanly
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not exit in time")
	}
}

func TestMultipleBreakpoints(t *testing.T) {
	srv, client := setupServer(t, `var a = 1;
var b = 2;
var c = 3;
var d = 4;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.setBreakpoints("", 2, 4)
	client.configurationDone()

	// First breakpoint at line 2
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}
	st := client.stackTrace()
	if st.Body.StackFrames[0].Line != 2 {
		t.Fatalf("Expected line 2, got %d", st.Body.StackFrames[0].Line)
	}

	// Continue to second breakpoint
	client.continueExec()
	stopped = client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}
	st = client.stackTrace()
	if st.Body.StackFrames[0].Line != 4 {
		t.Fatalf("Expected line 4, got %d", st.Body.StackFrames[0].Line)
	}

	// Continue to end
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPConditionalBreakpoint(t *testing.T) {
	srv, client := setupServer(t, `var x = 0;
for (var i = 0; i < 5; i++) {
  x = i;
}`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	// Set conditional breakpoint: only pause when i == 3
	client.setBreakpointsWithOpts("", dap.SourceBreakpoint{
		Line:      3,
		Condition: "i == 3",
	})
	client.configurationDone()

	// Should hit breakpoint only when i == 3
	client.expectEvent("stopped")
	st := client.stackTrace()
	if st.Body.StackFrames[0].Line != 3 {
		t.Fatalf("Expected line 3, got %d", st.Body.StackFrames[0].Line)
	}

	// Verify i == 3
	evalResp := client.evaluate("i", 1)
	if evalResp.Body.Result != "3" {
		t.Fatalf("Expected i=3, got i=%s", evalResp.Body.Result)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPHitCountBreakpoint(t *testing.T) {
	srv, client := setupServer(t, `var x = 0;
for (var i = 0; i < 10; i++) {
  x = i;
}`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	// Hit count: pause every 3rd hit
	client.setBreakpointsWithOpts("", dap.SourceBreakpoint{
		Line:         3,
		HitCondition: "3",
	})
	client.configurationDone()

	// First pause: 3rd hit (i=2)
	client.expectEvent("stopped")
	evalResp := client.evaluate("i", 1)
	if evalResp.Body.Result != "2" {
		t.Fatalf("Expected i=2 at 3rd hit, got i=%s", evalResp.Body.Result)
	}

	// Continue → 6th hit (i=5)
	client.continueExec()
	client.expectEvent("stopped")
	evalResp = client.evaluate("i", 1)
	if evalResp.Body.Result != "5" {
		t.Fatalf("Expected i=5 at 6th hit, got i=%s", evalResp.Body.Result)
	}

	// Continue → 9th hit (i=8)
	client.continueExec()
	client.expectEvent("stopped")
	evalResp = client.evaluate("i", 1)
	if evalResp.Body.Result != "8" {
		t.Fatalf("Expected i=8 at 9th hit, got i=%s", evalResp.Body.Result)
	}

	// Continue to end
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPLogPoint(t *testing.T) {
	srv, client := setupServer(t, `var a = 10;
var b = 20;
var c = 30;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	// Log point on line 2: should log instead of pausing
	client.setBreakpointsWithOpts("", dap.SourceBreakpoint{
		Line:       2,
		LogMessage: "value of a is {a}",
	})
	client.configurationDone()

	// Should NOT get a stopped event — should get an output event then terminated
	msg := client.read()
	switch m := msg.(type) {
	case *dap.OutputEvent:
		if m.Body.Category != "console" {
			t.Fatalf("Expected category 'console', got '%s'", m.Body.Category)
		}
		if m.Body.Output != "value of a is 10\n" {
			t.Fatalf("Expected 'value of a is 10\\n', got '%s'", m.Body.Output)
		}
	case *dap.StoppedEvent:
		t.Fatal("Log point should not cause a stopped event")
	default:
		t.Fatalf("Expected output event, got %T", msg)
	}

	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPExceptionBreakpoints(t *testing.T) {
	srv, client := setupServer(t, `var x = 1;
try {
  throw new Error("test error");
} catch(e) {
  x = 2;
}
var y = 3;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	// Set exception breakpoints for all exceptions
	client.setExceptionBreakpoints("all")
	client.configurationDone()

	// Should pause on the thrown exception
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "exception" {
		t.Fatalf("Expected reason 'exception', got '%s'", stopped.Body.Reason)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPExceptionBreakpointsUncaughtOnly(t *testing.T) {
	srv, client := setupServer(t, `var x = 1;
try {
  throw new Error("caught");
} catch(e) {
  x = 2;
}
var y = 3;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	// Only uncaught exceptions
	client.setExceptionBreakpoints("uncaught")
	client.configurationDone()

	// The exception is caught, so no stopped event — goes straight to terminated
	client.expectEvent("terminated")
	client.disconnect()
}

func TestDAPSetVariable(t *testing.T) {
	srv, client := setupServer(t, `var x = 10;
debugger;
var result = x * 2;`)

	go srv.Run()

	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Hit debugger statement
	client.expectEvent("stopped")

	// Get scopes to find the variable reference
	st := client.stackTrace()
	frameId := st.Body.StackFrames[0].Id
	scopesResp := client.scopes(frameId)

	// Find the scope containing x
	var scopeRef int
	for _, scope := range scopesResp.Body.Scopes {
		varsResp := client.variables(scope.VariablesReference)
		for _, v := range varsResp.Body.Variables {
			if v.Name == "x" && v.Value == "10" {
				scopeRef = scope.VariablesReference
				break
			}
		}
		if scopeRef != 0 {
			break
		}
	}
	if scopeRef == 0 {
		t.Fatal("Could not find variable x in any scope")
	}

	// Set x to 42
	setResp := client.setVariable(scopeRef, "x", "42")
	if setResp.Body.Value != "42" {
		t.Fatalf("Expected set value '42', got '%s'", setResp.Body.Value)
	}

	// Verify the change via eval
	evalResp := client.evaluate("x", frameId)
	if evalResp.Body.Result != "42" {
		t.Fatalf("Expected x=42 after set, got x=%s", evalResp.Body.Result)
	}

	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()
}

func TestListenTCP(t *testing.T) {
	r := goja.New()
	session, err := ListenTCP(r, "127.0.0.1:0", func() error {
		_, err := r.RunString("var x = 1;\ndebugger;\nvar y = 2;")
		return err
	})
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	defer session.Close()

	// Connect to the server
	conn, err := net.Dial("tcp", session.Addr.String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := newTestClient(t, conn, conn)

	// Run a minimal debug session
	client.initialize()
	client.launch(false)
	client.configurationDone()

	// Should hit debugger statement
	stopped := client.expectEvent("stopped").(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("Expected 'breakpoint', got '%s'", stopped.Body.Reason)
	}

	// Continue to end
	client.continueExec()
	client.expectEvent("terminated")
	client.disconnect()

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		t.Fatalf("Session error: %v", err)
	}
}
