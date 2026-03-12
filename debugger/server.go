package debugger

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/dop251/goja"
	dap "github.com/google/go-dap"
)

// Server is a DAP (Debug Adapter Protocol) server for a goja Runtime.
// It bridges DAP messages from an IDE (e.g., VS Code) to goja's debug API.
type Server struct {
	runtime  *goja.Runtime
	debugger *goja.Debugger
	reader   *bufio.Reader
	writer   io.Writer
	runFunc  func() error

	// Outgoing message sequencing
	sendMu sync.Mutex
	seq    int

	// Reference manager (server-goroutine only)
	refs *RefManager

	// Hook bridge channels
	stoppedCh    chan stopInfo         // VM → Server: VM paused
	inspectCh    chan inspectRequest   // Server → VM: inspect while paused
	resumeCh     chan goja.DebugAction // Server → VM: resume
	logCh        chan logEntry         // VM → Server: log point output
	disconnectCh chan struct{}         // closed on disconnect to unblock debugHook

	// Session state
	configured  bool
	launched    bool
	stopOnEntry bool
	paused      bool
	terminated  bool
	vmDone      bool

	// Source path mapping: basename → full path from IDE.
	// Populated when setBreakpoints sends an absolute path;
	// used by onStackTrace to return paths the IDE can open.
	sourcePathMap map[string]string

	// Termination
	doneCh chan error // VM goroutine signals completion
}

type stopInfo struct {
	event goja.DebugEvent
	pos   goja.DebugPosition
}

type inspectRequest struct {
	fn       func(ctx *goja.DebugContext) interface{}
	resultCh chan<- interface{}
}

type evalResult struct {
	val goja.Value
	err error
}

type logEntry struct {
	message string
	pos     goja.DebugPosition
}

type dapMessage struct {
	msg dap.Message
	err error
}

// NewServer creates a new DAP debug adapter server.
// reader/writer are the DAP transport (typically stdin/stdout or a TCP connection).
// runFunc is called to start JS execution after ConfigurationDone.
// It runs on a new goroutine and should execute JS code via the runtime.
func NewServer(r *goja.Runtime, reader io.Reader, writer io.Writer, runFunc func() error) *Server {
	return &Server{
		runtime:       r,
		reader:        bufio.NewReader(reader),
		writer:        writer,
		runFunc:       runFunc,
		refs:          NewRefManager(),
		stoppedCh:     make(chan stopInfo, 1),
		inspectCh:     make(chan inspectRequest),
		resumeCh:      make(chan goja.DebugAction),
		logCh:         make(chan logEntry, 16),
		doneCh:        make(chan error, 1),
		disconnectCh:  make(chan struct{}),
		sourcePathMap: make(map[string]string),
	}
}

// Run starts the DAP message loop. It blocks until disconnect or error.
func (s *Server) Run() error {
	s.debugger = goja.NewDebugger(s.debugHook)
	s.debugger.SetLogHook(func(msg string, pos goja.DebugPosition) {
		s.logCh <- logEntry{message: msg, pos: pos}
	})
	s.runtime.SetDebugger(s.debugger)

	// Detach debugger only after the VM goroutine has finished,
	// to avoid racing with VM goroutine reads of vm.dbg.
	// Also respect disconnectCh so we don't block if the client
	// disconnects before runFunc completes.
	defer func() {
		if s.configured && !s.vmDone {
			select {
			case <-s.doneCh:
			case <-s.disconnectCh:
			}
		}
		s.runtime.SetDebugger(nil)
	}()

	msgCh := make(chan dapMessage, 1)
	go func() {
		for {
			msg, err := dap.ReadProtocolMessage(s.reader)
			msgCh <- dapMessage{msg, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case dm := <-msgCh:
			if dm.err != nil {
				return dm.err
			}
			if done := s.handleMessage(dm.msg); done {
				return nil
			}

		case stop := <-s.stoppedCh:
			s.refs.Clear()
			reason := eventToReason(stop.event)
			s.send(&dap.StoppedEvent{
				Event: s.newEvent("stopped"),
				Body: dap.StoppedEventBody{
					Reason:            reason,
					ThreadId:          1,
					AllThreadsStopped: true,
				},
			})
			s.paused = true

		case entry := <-s.logCh:
			s.send(&dap.OutputEvent{
				Event: s.newEvent("output"),
				Body: dap.OutputEventBody{
					Category: "console",
					Output:   entry.message + "\n",
					Source: &dap.Source{
						Path: entry.pos.Filename,
					},
					Line: entry.pos.Line,
				},
			})

		case err := <-s.doneCh:
			_ = err
			s.send(&dap.TerminatedEvent{Event: s.newEvent("terminated")})
			s.paused = false
			s.terminated = true
			s.vmDone = true
		}
	}
}

// debugHook is the debug hook called on the VM goroutine when the VM pauses.
// It acts as a bridge: notifies the server goroutine, then blocks processing
// inspection requests until a resume action is received.
func (s *Server) debugHook(ctx *goja.DebugContext, event goja.DebugEvent, pos goja.DebugPosition) goja.DebugAction {
	// Notify server goroutine that VM is paused
	select {
	case s.stoppedCh <- stopInfo{event: event, pos: pos}:
	case <-s.disconnectCh:
		return goja.DebugContinue
	}

	// Block in select loop: process inspect requests until resume
	for {
		select {
		case req := <-s.inspectCh:
			result := req.fn(ctx) // Execute on VM goroutine with valid DebugContext
			req.resultCh <- result
		case action := <-s.resumeCh:
			return action
		case <-s.disconnectCh:
			return goja.DebugContinue
		}
	}
}

// inspect executes a function on the VM goroutine while it is paused.
// Returns (result, true) on success, or (nil, false) if the VM is not paused.
func (s *Server) inspect(fn func(ctx *goja.DebugContext) interface{}) (interface{}, bool) {
	if !s.paused {
		return nil, false
	}
	resultCh := make(chan interface{}, 1)
	s.inspectCh <- inspectRequest{fn: fn, resultCh: resultCh}
	return <-resultCh, true
}

// send writes a DAP message to the transport with proper sequencing.
func (s *Server) send(msg dap.Message) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	s.seq++
	setSeq(msg, s.seq)
	_ = dap.WriteProtocolMessage(s.writer, msg)
}

// newResponse creates a success Response base for the given request.
func (s *Server) newResponse(requestSeq int, command string) dap.Response {
	return dap.Response{
		ProtocolMessage: dap.ProtocolMessage{Type: "response"},
		RequestSeq:      requestSeq,
		Command:         command,
		Success:         true,
	}
}

// newEvent creates an Event base with the given event name.
func (s *Server) newEvent(event string) dap.Event {
	return dap.Event{
		ProtocolMessage: dap.ProtocolMessage{Type: "event"},
		Event:           event,
	}
}

// sendError sends an error response for a request.
func (s *Server) sendError(requestSeq int, command string, message string) {
	s.send(&dap.ErrorResponse{
		Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{Type: "response"},
			RequestSeq:      requestSeq,
			Command:         command,
			Success:         false,
			Message:         message,
		},
	})
}

// eventToReason maps a goja DebugEvent to a DAP stopped reason string.
func eventToReason(event goja.DebugEvent) string {
	switch event {
	case goja.DebugEventBreakpoint:
		return "breakpoint"
	case goja.DebugEventStep:
		return "step"
	case goja.DebugEventPause:
		return "pause"
	case goja.DebugEventDebuggerStmt:
		return "breakpoint"
	case goja.DebugEventEntry:
		return "entry"
	case goja.DebugEventException:
		return "exception"
	default:
		return "unknown"
	}
}

// setSeq sets the Seq field on any DAP message.
// All go-dap Response/Event types implement ResponseMessage/EventMessage,
// giving access to the embedded ProtocolMessage.Seq field.
func setSeq(msg dap.Message, seq int) {
	switch m := msg.(type) {
	case dap.ResponseMessage:
		m.GetResponse().Seq = seq
	case dap.EventMessage:
		m.GetEvent().Seq = seq
	}
}

// ServeTCP starts a DAP server listening on the given TCP address.
// It blocks until a single client connects, runs a debug session, then returns.
// The addr parameter is a TCP address (e.g., "127.0.0.1:4711" or ":4711").
// runFunc is called to start JS execution after the client sends ConfigurationDone.
//
// Example usage:
//
//	r := goja.New()
//	err := debugger.ServeTCP(r, "127.0.0.1:4711", func() error {
//	    _, err := r.RunString(script)
//	    return err
//	})
func ServeTCP(r *goja.Runtime, addr string, runFunc func() error) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("debugger: listen on %s: %w", addr, err)
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("debugger: accept: %w", err)
	}
	defer conn.Close()

	srv := NewServer(r, conn, conn, runFunc)
	return srv.Run()
}

// TCPSession represents a running DAP debug session over TCP.
// Use Wait() to block until the session completes.
type TCPSession struct {
	// Addr is the address the server is listening on. Useful when the
	// port was auto-assigned (addr ":0").
	Addr  net.Addr
	ln    net.Listener
	errCh chan error
}

// Wait blocks until the debug session completes and returns any error.
func (s *TCPSession) Wait() error {
	return <-s.errCh
}

// Close stops the listener, which will cause the session to end if
// no client has connected yet. If a session is in progress, it
// interrupts it.
func (s *TCPSession) Close() error {
	return s.ln.Close()
}

// ListenTCP starts a DAP server listening on the given TCP address in
// the background. It returns immediately with a TCPSession that can be
// used to get the listening address and wait for session completion.
//
// The server accepts a single client connection. Once the client
// disconnects, the session ends.
//
// This is designed for embedding: start the listener, print the port,
// then call session.Wait() to block until debugging is done.
//
// Example:
//
//	r := goja.New()
//	session, err := debugger.ListenTCP(r, "127.0.0.1:0", func() error {
//	    _, err := r.RunString(script)
//	    return err
//	})
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Debugger listening on %s\n", session.Addr)
//	if err := session.Wait(); err != nil { log.Fatal(err) }
func ListenTCP(r *goja.Runtime, addr string, runFunc func() error) (*TCPSession, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("debugger: listen on %s: %w", addr, err)
	}

	session := &TCPSession{
		Addr:  ln.Addr(),
		ln:    ln,
		errCh: make(chan error, 1),
	}

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				session.errCh <- fmt.Errorf("debugger: accept: %w", err)
				return
			}

			// Detect probe connections (e.g. port-readiness checks) by
			// peeking at the first byte. Real DAP clients send an
			// Initialize request promptly; probes connect and disconnect.
			br := bufio.NewReader(conn)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			if _, err := br.Peek(1); err != nil {
				conn.Close()
				continue // probe — re-accept
			}
			conn.SetReadDeadline(time.Time{}) // clear deadline

			srv := NewServer(r, br, conn, runFunc)
			session.errCh <- srv.Run()
			conn.Close()
			return
		}
	}()

	return session, nil
}

// AttachSession is a debug session where the debugger attaches to an existing
// runtime. Unlike ListenTCP/ServeTCP (which own the JS execution via runFunc),
// AttachSession lets the caller manage execution separately — the debugger
// intercepts all JS execution on the runtime until the session ends.
type AttachSession struct {
	*TCPSession
	readyCh   chan struct{}
	closeCh   chan struct{}
	closeOnce sync.Once
}

// Ready blocks until a DAP client has connected and sent ConfigurationDone
// (i.e., breakpoints are set and the runtime is ready for JS execution).
func (s *AttachSession) Ready() {
	<-s.readyCh
}

// Close signals the debug session to terminate and waits for cleanup.
// It is safe to call multiple times.
func (s *AttachSession) Close() error {
	s.closeOnce.Do(func() { close(s.closeCh) })
	return s.TCPSession.Wait()
}

// AttachTCP starts a DAP debug server that attaches to an existing runtime.
// Unlike ListenTCP, there is no runFunc — the caller manages JS execution
// separately (e.g., via runtime.RunString). The debugger intercepts all JS
// execution on the runtime until the session ends.
//
// Call session.Ready() to block until a client connects and configures
// breakpoints, then execute JS normally. Call session.Close() when done.
//
// Example:
//
//	r := goja.New()
//	session, err := debugger.AttachTCP(r, "127.0.0.1:0")
//	if err != nil { log.Fatal(err) }
//	fmt.Printf("Debugger on %s\n", session.Addr)
//	session.Ready()  // wait for VS Code to connect
//	r.RunString(script)  // breakpoints work
//	session.Close()
func AttachTCP(r *goja.Runtime, addr string) (*AttachSession, error) {
	as := &AttachSession{
		readyCh: make(chan struct{}),
		closeCh: make(chan struct{}),
	}

	session, err := ListenTCP(r, addr, func() error {
		close(as.readyCh)
		<-as.closeCh
		return nil
	})
	if err != nil {
		return nil, err
	}
	as.TCPSession = session

	// If the session ends before Close() (e.g., client disconnect),
	// unblock the runFunc goroutine to prevent a leak.
	go func() {
		session.Wait()
		as.closeOnce.Do(func() { close(as.closeCh) })
	}()

	return as, nil
}
