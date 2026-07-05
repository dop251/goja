package debugger

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	dap "github.com/google/go-dap"
)

// normalizeSourcePath extracts the local filesystem path from a
// vscode-remote URI. In WSL/SSH environments VS Code sends source paths
// like "vscode-remote://wsl+distro/home/user/file.ts"; the DAP server
// must store and return plain filesystem paths so VS Code doesn't
// double-wrap them.
func normalizeSourcePath(p string) string {
	// Handle URI scheme (vscode-remote://wsl+distro/path)
	if strings.Contains(p, "://") {
		if u, err := url.Parse(p); err == nil && u.Path != "" {
			return u.Path
		}
	}
	return p
}

// handleMessage dispatches a DAP message to the appropriate handler.
// Returns true if the server should exit the Run() loop.
func (s *Server) handleMessage(msg dap.Message) bool {
	// After termination, only accept Disconnect
	if s.terminated {
		if req, ok := msg.(*dap.DisconnectRequest); ok {
			return s.onDisconnect(req)
		}
		if req, ok := msg.(dap.RequestMessage); ok {
			r := req.GetRequest()
			s.sendError(r.Seq, r.Command, "Program has terminated")
		}
		return false
	}

	switch req := msg.(type) {
	case *dap.InitializeRequest:
		s.onInitialize(req)
	case *dap.LaunchRequest:
		s.onLaunch(req)
	case *dap.AttachRequest:
		s.onAttach(req)
	case *dap.SetBreakpointsRequest:
		s.onSetBreakpoints(req)
	case *dap.SetExceptionBreakpointsRequest:
		s.onSetExceptionBreakpoints(req)
	case *dap.ConfigurationDoneRequest:
		s.onConfigurationDone(req)
	case *dap.ThreadsRequest:
		s.onThreads(req)
	case *dap.StackTraceRequest:
		s.onStackTrace(req)
	case *dap.ScopesRequest:
		s.onScopes(req)
	case *dap.VariablesRequest:
		s.onVariables(req)
	case *dap.EvaluateRequest:
		s.onEvaluate(req)
	case *dap.SetVariableRequest:
		s.onSetVariable(req)
	case *dap.ContinueRequest:
		s.onContinue(req)
	case *dap.NextRequest:
		s.onNext(req)
	case *dap.StepInRequest:
		s.onStepIn(req)
	case *dap.StepOutRequest:
		s.onStepOut(req)
	case *dap.PauseRequest:
		s.onPause(req)
	case *dap.DisconnectRequest:
		return s.onDisconnect(req)
	case *dap.TerminateRequest:
		s.onTerminate(req)
	default:
		if req, ok := msg.(dap.RequestMessage); ok {
			r := req.GetRequest()
			s.sendError(r.Seq, r.Command, fmt.Sprintf("Unsupported request: %s", r.Command))
		}
	}
	return false
}

func (s *Server) onInitialize(req *dap.InitializeRequest) {
	s.send(&dap.InitializeResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body: dap.Capabilities{
			SupportsConfigurationDoneRequest:          true,
			SupportsEvaluateForHovers:                 true,
			SupportsTerminateRequest:                  true,
			SupportsConditionalBreakpoints:             true,
			SupportsHitConditionalBreakpoints:          true,
			SupportsLogPoints:                          true,
			SupportsSetVariable:                        true,
			ExceptionBreakpointFilters: []dap.ExceptionBreakpointsFilter{
				{Filter: "all", Label: "All Exceptions", Description: "Break on all exceptions"},
				{Filter: "uncaught", Label: "Uncaught Exceptions", Description: "Break on uncaught exceptions", Default: true},
			},
		},
	})
	s.send(&dap.InitializedEvent{Event: s.newEvent("initialized")})
}

func (s *Server) onLaunch(req *dap.LaunchRequest) {
	var args map[string]interface{}
	if err := json.Unmarshal(req.Arguments, &args); err == nil {
		s.stopOnEntry, _ = args["stopOnEntry"].(bool)
	}
	s.launched = true
	s.send(&dap.LaunchResponse{Response: s.newResponse(req.Seq, req.Command)})
}

func (s *Server) onAttach(req *dap.AttachRequest) {
	s.launched = true
	s.send(&dap.AttachResponse{Response: s.newResponse(req.Seq, req.Command)})
}

func (s *Server) onSetBreakpoints(req *dap.SetBreakpointsRequest) {
	source := req.Arguments.Source
	path := normalizeSourcePath(source.Path)
	if path == "" {
		path = source.Name
	}

	// Remember the local filesystem path for this source basename so we
	// can map goja's short source names back to paths the IDE can open.
	if filepath.IsAbs(path) {
		s.sourcePathMap[filepath.Base(path)] = path
	}

	s.debugger.ClearBreakpoints(path)

	bps := make([]dap.Breakpoint, len(req.Arguments.Breakpoints))
	for i, sbp := range req.Arguments.Breakpoints {
		var opts []goja.BreakpointOption
		if sbp.Condition != "" {
			opts = append(opts, goja.WithCondition(sbp.Condition))
		}
		if sbp.HitCondition != "" {
			opts = append(opts, goja.WithHitCondition(sbp.HitCondition))
		}
		if sbp.LogMessage != "" {
			opts = append(opts, goja.WithLogMessage(sbp.LogMessage))
		}
		bp := s.debugger.SetBreakpoint(path, sbp.Line, 0, opts...)
		bps[i] = dap.Breakpoint{
			Id:       bp.ID,
			Verified: true,
			Source:   &dap.Source{Name: filepath.Base(path), Path: path},
			Line:     sbp.Line,
		}
	}
	s.send(&dap.SetBreakpointsResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body:     dap.SetBreakpointsResponseBody{Breakpoints: bps},
	})
}

func (s *Server) onSetExceptionBreakpoints(req *dap.SetExceptionBreakpointsRequest) {
	s.debugger.SetExceptionBreakpoints(req.Arguments.Filters)
	s.send(&dap.SetExceptionBreakpointsResponse{
		Response: s.newResponse(req.Seq, req.Command),
	})
}

func (s *Server) onConfigurationDone(req *dap.ConfigurationDoneRequest) {
	s.configured = true
	s.send(&dap.ConfigurationDoneResponse{Response: s.newResponse(req.Seq, req.Command)})

	if s.stopOnEntry {
		s.runtime.RequestPause()
	}

	// Start VM on separate goroutine
	go func() {
		err := s.runFunc()
		s.doneCh <- err
	}()
}

func (s *Server) onThreads(req *dap.ThreadsRequest) {
	s.send(&dap.ThreadsResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body: dap.ThreadsResponseBody{
			Threads: []dap.Thread{{Id: 1, Name: "main"}},
		},
	})
}

func (s *Server) onStackTrace(req *dap.StackTraceRequest) {
	result, ok := s.inspect(func(ctx *goja.DebugContext) interface{} {
		return ctx.CallStack()
	})
	if !ok {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	gojaFrames := result.([]goja.StackFrame)

	frames := make([]dap.StackFrame, len(gojaFrames))
	for i, f := range gojaFrames {
		pos := f.Position()
		name := f.FuncName()
		if name == "" {
			name = "<anonymous>"
		}
		// Resolve the source path: if goja has a short name (e.g. "fibonacci.ts"),
		// map it to the full path the IDE knows about.
		sourcePath := pos.Filename
		if !filepath.IsAbs(sourcePath) {
			if fullPath, ok := s.sourcePathMap[filepath.Base(sourcePath)]; ok {
				sourcePath = fullPath
			}
		}
		frames[i] = dap.StackFrame{
			Id:   i + 1, // 1-based frame IDs
			Name: name,
			Source: &dap.Source{
				Name: filepath.Base(sourcePath),
				Path: sourcePath,
			},
			Line:   pos.Line,
			Column: pos.Column,
		}
	}
	s.send(&dap.StackTraceResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body: dap.StackTraceResponseBody{
			StackFrames: frames,
			TotalFrames: len(frames),
		},
	})
}

func (s *Server) onScopes(req *dap.ScopesRequest) {
	frameIndex := req.Arguments.FrameId - 1 // Convert 1-based to 0-based
	result, ok := s.inspect(func(ctx *goja.DebugContext) interface{} {
		return ctx.Scopes(frameIndex)
	})
	if !ok {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	gojaScopes := result.([]goja.DebugScope)

	scopes := make([]dap.Scope, len(gojaScopes))
	for i, gs := range gojaScopes {
		ref := s.refs.AddScope(frameIndex, i, gs)
		scopes[i] = dap.Scope{
			Name:               gs.Name,
			VariablesReference: ref,
			Expensive:          gs.Type == "global",
		}
	}
	s.send(&dap.ScopesResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body:     dap.ScopesResponseBody{Scopes: scopes},
	})
}

func (s *Server) onVariables(req *dap.VariablesRequest) {
	ref := req.Arguments.VariablesReference
	entry, ok := s.refs.Get(ref)
	if !ok {
		s.sendError(req.Seq, req.Command, "Invalid variable reference")
		return
	}

	var vars []dap.Variable
	switch v := entry.(type) {
	case scopeEntry:
		for _, dv := range v.scope.Variables {
			vars = append(vars, s.valueToVariable(dv.Name, dv.Value))
		}
	case objectEntry:
		result, inspectOk := s.inspect(func(ctx *goja.DebugContext) interface{} {
			obj := v.object
			var props []dap.Variable
			for _, key := range obj.Keys() {
				val := obj.Get(key)
				props = append(props, s.valueToVariable(key, val))
			}
			return props
		})
		if !inspectOk {
			s.sendError(req.Seq, req.Command, "VM is not paused")
			return
		}
		vars = result.([]dap.Variable)
	}

	if vars == nil {
		vars = []dap.Variable{}
	}

	s.send(&dap.VariablesResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body:     dap.VariablesResponseBody{Variables: vars},
	})
}

func (s *Server) onEvaluate(req *dap.EvaluateRequest) {
	frameIndex := 0
	if req.Arguments.FrameId > 0 {
		frameIndex = req.Arguments.FrameId - 1
	}

	result, ok := s.inspect(func(ctx *goja.DebugContext) interface{} {
		val, err := ctx.Eval(frameIndex, req.Arguments.Expression)
		return evalResult{val, err}
	})
	if !ok {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	er := result.(evalResult)
	if er.err != nil {
		s.sendError(req.Seq, req.Command, er.err.Error())
		return
	}

	resp := &dap.EvaluateResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body: dap.EvaluateResponseBody{
			Result: er.val.String(),
		},
	}
	if obj, ok := er.val.(*goja.Object); ok {
		resp.Body.VariablesReference = s.refs.AddObject(obj)
	}
	s.send(resp)
}

func (s *Server) onSetVariable(req *dap.SetVariableRequest) {
	ref := req.Arguments.VariablesReference
	entry, ok := s.refs.Get(ref)
	if !ok {
		s.sendError(req.Seq, req.Command, "Invalid variable reference")
		return
	}

	result, inspectOk := s.inspect(func(ctx *goja.DebugContext) interface{} {
		// Evaluate the new value expression in the target frame's scope,
		// not always frame 0, so identifiers resolve correctly.
		frameIndex := 0
		if se, ok := entry.(scopeEntry); ok {
			frameIndex = se.frameIndex
		}
		newVal, err := ctx.Eval(frameIndex, req.Arguments.Value)
		if err != nil {
			return evalResult{err: err}
		}

		switch e := entry.(type) {
		case scopeEntry:
			err = ctx.SetVariable(e.frameIndex, e.scopeIndex, req.Arguments.Name, newVal)
		case objectEntry:
			err = e.object.Set(req.Arguments.Name, newVal)
		}
		if err != nil {
			return evalResult{err: err}
		}
		return evalResult{val: newVal}
	})
	if !inspectOk {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	er := result.(evalResult)
	if er.err != nil {
		s.sendError(req.Seq, req.Command, er.err.Error())
		return
	}

	resp := &dap.SetVariableResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body: dap.SetVariableResponseBody{
			Value: er.val.String(),
		},
	}
	if obj, ok := er.val.(*goja.Object); ok {
		resp.Body.VariablesReference = s.refs.AddObject(obj)
	}
	s.send(resp)
}

func (s *Server) onContinue(req *dap.ContinueRequest) {
	if !s.paused {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	s.send(&dap.ContinueResponse{
		Response: s.newResponse(req.Seq, req.Command),
		Body:     dap.ContinueResponseBody{AllThreadsContinued: true},
	})
	s.paused = false
	s.resumeCh <- goja.DebugContinue
}

func (s *Server) onNext(req *dap.NextRequest) {
	if !s.paused {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	s.send(&dap.NextResponse{Response: s.newResponse(req.Seq, req.Command)})
	s.paused = false
	s.resumeCh <- goja.DebugStepOver
}

func (s *Server) onStepIn(req *dap.StepInRequest) {
	if !s.paused {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	s.send(&dap.StepInResponse{Response: s.newResponse(req.Seq, req.Command)})
	s.paused = false
	s.resumeCh <- goja.DebugStepIn
}

func (s *Server) onStepOut(req *dap.StepOutRequest) {
	if !s.paused {
		s.sendError(req.Seq, req.Command, "VM is not paused")
		return
	}
	s.send(&dap.StepOutResponse{Response: s.newResponse(req.Seq, req.Command)})
	s.paused = false
	s.resumeCh <- goja.DebugStepOut
}

func (s *Server) onPause(req *dap.PauseRequest) {
	s.runtime.RequestPause()
	s.send(&dap.PauseResponse{Response: s.newResponse(req.Seq, req.Command)})
}

func (s *Server) onTerminate(req *dap.TerminateRequest) {
	s.runtime.Interrupt("terminated by debugger")
	s.send(&dap.TerminateResponse{Response: s.newResponse(req.Seq, req.Command)})
}

func (s *Server) onDisconnect(req *dap.DisconnectRequest) bool {
	// Close disconnectCh to unblock debugHook regardless of whether
	// the stopped event has been processed yet (avoids deadlock when
	// disconnect races with a pending stop notification).
	close(s.disconnectCh)
	s.paused = false
	s.runtime.Interrupt("debugger disconnected")
	s.send(&dap.DisconnectResponse{Response: s.newResponse(req.Seq, req.Command)})
	return true
}

// valueToVariable converts a goja Value to a DAP Variable.
func (s *Server) valueToVariable(name string, val goja.Value) dap.Variable {
	if val == nil {
		return dap.Variable{Name: name, Value: "undefined", Type: "undefined"}
	}

	v := dap.Variable{
		Name: name,
	}

	switch {
	case goja.IsNull(val):
		v.Type = "null"
		v.Value = "null"
	case goja.IsUndefined(val):
		v.Type = "undefined"
		v.Value = "undefined"
	case goja.IsNaN(val):
		v.Type = "number"
		v.Value = "NaN"
	case goja.IsInfinity(val):
		v.Type = "number"
		v.Value = "Infinity"
	default:
		if obj, ok := val.(*goja.Object); ok {
			v.VariablesReference = s.refs.AddObject(obj)
			// Use ClassName() which is safe (no JS execution) instead of
			// String() which calls toString() and can corrupt VM state.
			cls := obj.ClassName()
			switch cls {
			case "Function":
				v.Type = "function"
				v.Value = "function"
			case "Array":
				v.Type = "array"
				v.Value = cls
			default:
				v.Type = "object"
				v.Value = cls
			}
		} else {
			// Primitives: String() is safe (no JS execution for int, float, bool, string).
			v.Value = val.String()
			if exportType := val.ExportType(); exportType != nil {
				v.Type = exportType.Kind().String()
			}
		}
	}

	return v
}
