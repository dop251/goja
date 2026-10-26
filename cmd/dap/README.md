# goja DAP debugger

`goja-dap` is a [Debug Adapter Protocol](https://microsoft.github.io/debug-adapter-protocol/)
server built on top of the VM-level debugger in this repository (`debugger.go` +
`vm.debug()`). It lets you debug JavaScript running in goja from VS Code, or from
any other DAP client.

## Features

- Breakpoints (`setBreakpoints`)
- Stepping: step over / step in / step out (`next` / `stepIn` / `stepOut`)
- Call stack (`stackTrace`)
- Variable inspection: Local / Global scopes, with lazy expansion of objects
  (`scopes` / `variables`)
- Expression evaluation in the debug console (`evaluate`, executed in the
  context of the paused frame)
- `console.log` output forwarded as DAP `output` events
- `debugger;` statements trigger a pause

## Build

```sh
go build -o goja-dap ./cmd/dap
```

## Usage

### TCP mode (attach)

```sh
./goja-dap -port 5678 path/to/script.js
```

VS Code `launch.json` (with any DAP client extension, or "Debugger for goja"):

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "goja",
      "request": "attach",
      "name": "Attach to goja",
      "debugServer": 5678
    }
  ]
}
```

### stdio mode (editor-managed launch)

```sh
./goja-dap -stdio path/to/script.js
```

## Protocol sequence

```
client                       goja-dap                       goja VM (debug loop)
  |-- initialize ------------>|                                 |
  |<-- InitializeResponse + InitializedEvent                    |
  |-- attach ---------------->|                                 |
  |<-- AttachResponse         |                                 |
  |-- setBreakpoints -------->|  SetBreakpoint()                |
  |-- configurationDone ----->|  start(): go vm.RunScript() ---->| vm.debug()
  |                           |  pump: dbg.Continue() --------->| ...hit breakpoint...
  |<-- stopped(breakpoint) ---|<-------------------------------| activate()
  |-- stackTrace / scopes / variables / evaluate -------------->|
  |-- continue / next / stepIn / stepOut ---------------------->| resume
  |                           |  ...                            |
  |<-- terminated ------------|<-------------------------------| program end
```

## Implementation

| File | Purpose |
|---|---|
| `main.go` | Entry point: `-port` / `-stdio` / `-stopOnEntry`, DAP read/write loop |
| `adapter.go` | Request dispatch, breakpoints/stack/scopes/variables/evaluate, console forwarding, writer queue |
| `adapter_test.go` | In-process pipe smoke test (`go test ./...`) |
| `testdata/sample.js` | Smoke test script |

The VM-level debugger lives in `debugger.go` at the repository root
(`Runtime.AttachDebugger()` returns a `*goja.Debugger`). The hooks it relies on
are in `vm.go` (the `vm.debug()` loop and the `_debugger` instruction),
`compiler.go` (debug mode disables dynamic scope optimisations),
`compiler_stmt.go` (emits the `debugger` instruction in debug mode) and
`runtime.go` (`compileDebug`).

## Limitations

- One script per session; every new TCP connection replays the script from the
  beginning.
- `pause` (asynchronous interrupt) is not supported yet.
- Breakpoints are matched by "file base name + line number", so files with the
  same name in different directories may interfere with each other.
