# goja/debugger - DAP Debug Adapter for Goja

A [Debug Adapter Protocol](https://microsoft.github.io/debug-adapter-protocol/) (DAP) server that enables debugging JavaScript and TypeScript code running in the [goja](https://github.com/dop251/goja) runtime from VS Code and other DAP-compatible editors.

## Features

- **Breakpoints** — line breakpoints, conditional breakpoints, hit count breakpoints, log points
- **Exception breakpoints** — break on all exceptions or uncaught only
- **Stepping** — step over, step into, step out
- **Call stack** — full call stack with source-mapped positions
- **Variable inspection** — local, closure, and global scopes with object expansion
- **Set variable** — modify variable values while paused
- **Expression evaluation** — evaluate expressions in any stack frame context
- **Pause** — pause a running program at any time
- **`debugger` statement** — JavaScript `debugger` statements pause execution
- **Stop on entry** — optionally pause at the first statement
- **Source maps** — TypeScript debugging via `Program.SetSourceMap()`
- **Zero overhead** — no performance cost when debugger is not attached

## Quick Start

### 1. Install the VS Code extension

The extension is at `debugger/vscode-goja-debugger/`. A pre-built `.vsix` is
included in the repository for convenience. Install it in VS Code:

```bash
cd debugger/vscode-goja-debugger

# Option A (quickest): install the pre-built VSIX
code --install-extension vscode-goja-debugger-0.0.3.vsix

# Option B: symlink into VS Code extensions directory
ln -s "$(pwd)" ~/.vscode-server/extensions/vscode-goja-debugger

# Option C: package a fresh VSIX and install
npx @vscode/vsce package
code --install-extension vscode-goja-debugger-*.vsix
```

After installing, reload VS Code.

### 2. Add debugging to your Go application

Add a `--debug-port` flag to your application and start a DAP server when debugging is needed:

```go
package main

import (
    "flag"
    "fmt"
    "log"

    "github.com/dop251/goja"
    "github.com/dop251/goja/debugger"
)

func main() {
    debugPort := flag.Int("debug-port", 0, "DAP debug port (0 = no debugging)")
    flag.Parse()

    r := goja.New()

    // Your application's script
    script := `
        function fibonacci(n) {
            if (n <= 1) return n;
            return fibonacci(n - 1) + fibonacci(n - 2);
        }
        fibonacci(10);
    `

    if *debugPort > 0 {
        // Compile with debug info so all variables are visible
        p, err := goja.CompileForDebug("script.js", script, false)
        if err != nil {
            log.Fatal(err)
        }

        addr := fmt.Sprintf("127.0.0.1:%d", *debugPort)
        session, err := debugger.ListenTCP(r, addr, func() error {
            _, err := r.RunProgram(p)
            return err
        })
        if err != nil {
            log.Fatal(err)
        }
        fmt.Fprintf(os.Stderr, "Debugger listening on %s\n", session.Addr)
        if err := session.Wait(); err != nil {
            log.Fatal(err)
        }
    } else {
        // Normal execution — no debug overhead
        _, err := r.RunString(script)
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

### 3. Configure VS Code

Create `.vscode/launch.json` in your project:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug JS in my app",
      "type": "goja",
      "request": "launch",
      "program": "${workspaceFolder}",
      "args": ["--other-flags"],
      "port": 4711
    }
  ]
}
```

The `launch` request tells the extension to:

1. Run `go run . --port 4711 [args...]` in the `program` directory
2. Wait for the DAP server to start
3. Connect VS Code's debugger

Set breakpoints in your `.js` or `.ts` files and press **F5**.

## Compilation Modes

Goja supports two compilation modes. The debugger needs debug metadata (`dbgNames` maps) to display `let`/`const` variables and function parameters. Without it, only `var` declarations and closure variables are visible.

| Function                          | Debug info | Use case                             |
| --------------------------------- | ---------- | ------------------------------------ |
| `goja.Compile()`                  | No         | Production — zero overhead           |
| `goja.CompileForDebug()`          | Yes        | Pre-compiled programs for debugging  |
| `r.Compile()`                     | Auto       | Detects debugger at compile time     |
| `r.RunString()` / `r.RunScript()` | Auto       | Auto-detects if debugger is attached |

**Recommendation:** Use `r.Compile()` or `r.RunString()` — they automatically enable debug info when a debugger is attached, and skip it when not.

```go
// Auto-detect: debug info is generated only when r.SetDebugger() has been called
p, err := r.Compile("script.js", src, false)
```

## VS Code Launch Configurations

### Launch mode — extension builds and runs your Go app

```json
{
  "name": "Debug JS in goja",
  "type": "goja",
  "request": "launch",
  "program": "${workspaceFolder}/cmd/myapp",
  "args": ["--config", "dev.yaml"],
  "port": 4711,
  "buildArgs": ["-tags", "debug"],
  "cwd": "${workspaceFolder}",
  "env": { "DEBUG": "1" }
}
```

| Field       | Description                                               |
| ----------- | --------------------------------------------------------- |
| `program`   | Path to the Go package directory (must contain `main.go`) |
| `args`      | Arguments passed to the Go program after `--port`         |
| `port`      | TCP port for the DAP server (default: `4711`)             |
| `buildArgs` | Extra flags for `go run` (e.g., `-race`, `-tags`)         |
| `cwd`       | Working directory (defaults to `program` directory)       |
| `env`       | Extra environment variables                               |

The extension runs: `go run [buildArgs...] . --port PORT [args...]`

### Attach mode — connect to an already-running DAP server

```json
{
  "name": "Attach to goja",
  "type": "goja",
  "request": "attach",
  "port": 4711,
  "host": "127.0.0.1"
}
```

Start your Go application with debugging enabled first, then use this to connect.

### Compound mode — debug Go and JavaScript simultaneously

You can set breakpoints in both Go source files and JavaScript/TypeScript files at the same time using a compound launch configuration:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Go: my app",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/myapp",
      "args": ["--debug-port", "4711", "--script", "app.ts"]
    },
    {
      "name": "Attach to goja DAP",
      "type": "goja",
      "request": "attach",
      "port": 4711
    }
  ],
  "compounds": [
    {
      "name": "Debug Go + JavaScript",
      "configurations": ["Go: my app", "Attach to goja DAP"],
      "stopAll": true
    }
  ]
}
```

Select **"Debug Go + JavaScript"** from the debug dropdown and press F5. This starts two debug sessions:

1. **Go debugger (delve)** — set breakpoints in `.go` files, step through Go code
2. **Goja debugger (DAP)** — set breakpoints in `.js`/`.ts` files, inspect JS variables

The goja attach session automatically retries until the DAP server is ready, so it handles the startup timing gracefully.

## TypeScript / Source Map Support

If your code is transpiled (e.g., TypeScript via esbuild), attach a source map so breakpoints and stack traces reference the original source:

```go
import "github.com/go-sourcemap/sourcemap"

// Compile the transpiled JavaScript
p, _ := goja.CompileForDebug("app.ts", transpiledJS, false)

// Parse and attach the source map
sm, _ := sourcemap.Parse("", sourceMapJSON)
p.SetSourceMap(sm)

_, err := r.RunProgram(p)
```

Once attached, breakpoints set on `.ts` files resolve correctly through the source map.

## Go API Reference

### Debugger setup (in `goja` package)

```go
// Attach a debugger to a runtime.
r.SetDebugger(dbg)

// Request the VM to pause at the next statement (safe from any goroutine).
r.RequestPause()

// Compile with debug info for full variable visibility.
p, err := goja.CompileForDebug(name, src, strict)

// Compile with automatic debug detection (uses debug mode if debugger is attached).
p, err := r.Compile(name, src, strict)
```

### DAP server (in `goja/debugger` package)

```go
// Create a DAP server over any io.Reader/io.Writer (stdio, TCP, etc.)
srv := debugger.NewServer(runtime, reader, writer, runFunc)
err := srv.Run() // blocks until disconnect

// Listen on TCP, accept one client, run debug session.
session, err := debugger.ListenTCP(runtime, "127.0.0.1:4711", runFunc)
fmt.Println("Listening on", session.Addr)
err = session.Wait() // blocks until session ends
```

### Low-level debugger API (in `goja` package)

For custom integrations that don't use DAP:

```go
// Create a debugger with a custom hook
dbg := goja.NewDebugger(func(ctx *goja.DebugContext, event goja.DebugEvent, pos goja.DebugPosition) goja.DebugAction {
    fmt.Printf("Paused at %s:%d\n", pos.Filename, pos.Line)

    // Inspect the call stack
    for _, frame := range ctx.CallStack() {
        fmt.Printf("  %s at %s:%d\n", frame.FuncName(), frame.Position().Filename, frame.Position().Line)
    }

    // Inspect variables in the current frame
    for _, scope := range ctx.Scopes(0) {
        for _, v := range scope.Variables {
            fmt.Printf("  %s = %v\n", v.Name, v.Value)
        }
    }

    // Evaluate an expression in the current frame
    val, err := ctx.Eval(0, "myVar + 1")

    return goja.DebugActionContinue
})

// Set breakpoints
dbg.SetBreakpoint("script.js", 10, 0)
dbg.SetBreakpoint("script.js", 20, 0, goja.WithCondition("x > 5"))
dbg.SetBreakpoint("script.js", 30, 0, goja.WithHitCondition("3"))
dbg.SetBreakpoint("script.js", 40, 0, goja.WithLogMessage("x = {x}"))

// Exception breakpoints
dbg.SetExceptionBreakpoints([]string{"all"})     // break on all exceptions
dbg.SetExceptionBreakpoints([]string{"uncaught"}) // break on uncaught only

// Attach and run
r.SetDebugger(dbg)
r.RunString(script)
r.SetDebugger(nil) // detach when done
```

### Debug actions

The hook function returns a `DebugAction` that tells the VM what to do next:

| Action                | Behavior                                |
| --------------------- | --------------------------------------- |
| `DebugActionContinue` | Resume execution until next breakpoint  |
| `DebugActionStepOver` | Execute current line, stop at next line |
| `DebugActionStepInto` | Step into function calls                |
| `DebugActionStepOut`  | Run until the current function returns  |

### Debug events

The hook receives a `DebugEvent` indicating why execution paused:

| Event                         | Trigger                                                  |
| ----------------------------- | -------------------------------------------------------- |
| `DebugEventBreakpoint`        | Hit a breakpoint                                         |
| `DebugEventStep`              | Step operation completed                                 |
| `DebugEventDebuggerStatement` | `debugger` statement in JS                               |
| `DebugEventPause`             | `RequestPause()` was called                              |
| `DebugEventException`         | Exception thrown (when exception breakpoints are active) |

## Architecture

The DAP server uses a two-goroutine model:

```text
VS Code (client)          DAP Server goroutine          VM goroutine
      |                          |                           |
      |-- SetBreakpoints ------->|                           |
      |<-- Response -------------|                           |
      |                          |                           |
      |-- ConfigurationDone ---->|--- starts VM goroutine -->|
      |                          |                           |
      |                          |<-- stopped (breakpoint) --|  (hook blocks)
      |<-- StoppedEvent ---------|                           |
      |                          |                           |
      |-- StackTrace ----------->|--- inspect request ------>|
      |                          |<-- inspect response ------|
      |<-- Response -------------|                           |
      |                          |                           |
      |-- Continue ------------->|--- resume action -------->|  (hook returns)
      |                          |                           |-- continues
```

When the VM pauses, the debug hook blocks the VM goroutine. The server goroutine proxies inspection requests (StackTrace, Scopes, Variables, Evaluate, SetVariable) to the VM goroutine via channels, ensuring all VM state access is single-threaded.

## Supported DAP Requests

| Request                 | Status                                              |
| ----------------------- | --------------------------------------------------- |
| initialize              | Supported                                           |
| launch                  | Supported                                           |
| attach                  | Supported                                           |
| setBreakpoints          | Supported (line, conditional, hit count, log point) |
| setExceptionBreakpoints | Supported (`all`, `uncaught` filters)               |
| configurationDone       | Supported                                           |
| threads                 | Supported (single thread)                           |
| stackTrace              | Supported                                           |
| scopes                  | Supported                                           |
| variables               | Supported (with object expansion)                   |
| setVariable             | Supported                                           |
| evaluate                | Supported (in any frame context)                    |
| continue                | Supported                                           |
| next (step over)        | Supported                                           |
| stepIn                  | Supported                                           |
| stepOut                 | Supported                                           |
| pause                   | Supported                                           |
| terminate               | Supported                                           |
| disconnect              | Supported                                           |

## Thread Safety

- The `Server` manages all concurrency internally
- The `runFunc` is called on a new goroutine; all VM access should happen there
- Breakpoints can be set/cleared from any goroutine (`Debugger` uses `sync.RWMutex`)
- `RequestPause()` is safe to call from any goroutine
- After `Run()` returns, the debugger is detached and the runtime can be reused

## Limitations

- **Single thread** — goja is single-threaded; the server reports one thread
- **Eval side effects** — expression evaluation can modify program state (same as JavaScript `eval()`)
- **One debug session** — `ListenTCP` accepts one client connection at a time
