# Goja Debugger Examples

This directory contains examples showing how to use the Goja debugger for debugging JavaScript code.

## 📂 Examples

### 1. `hello_debugger/` - Quick Start (Beginner Friendly)
The simplest possible debugger example. Perfect for getting started in 30 seconds!

```bash
cd hello_debugger
go build -o hello_debugger main.go
./hello_debugger
```

**Features:**
- Step through code line by line
- See console.log output
- Simple breakpoint examples
- Debug JavaScript files from disk

### 2. `advanced/` - Full-Featured Debugger
A complete debugger with command-line interface and advanced features.

```bash
cd advanced
go build -o debugger main.go

# Run with breakpoints
./debugger -bp 5,10,15 test1.js

# Run in step mode
./debugger test1.js
```

**Features:**
- Command-line breakpoint specification
- Multiple stepping modes (into, over, out)
- Call stack visualization
- Source code display with context
- Interactive commands

## 🚀 Getting Started

If you're new to the Goja debugger:

1. Start with `hello_debugger/` - it's the easiest way to understand the basics
2. Once comfortable, explore `advanced/` for more powerful debugging features

## 🔧 Key Concepts

- **Breakpoints**: Pause execution at specific lines
- **Step Mode**: Execute code line by line
- **Debug Handler**: Your function that controls what happens when paused
- **Debug Commands**: Control execution flow (continue, step, etc.)

## 📖 API Overview

```go
// Enable debugger
debugger := vm.EnableDebugger()

// Set handler for pauses
debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
    // Your logic here
    return goja.DebugContinue
})

// Add breakpoints
debugger.AddBreakpoint(filename, line, column)

// Enable step mode
debugger.SetStepMode(true)
```

## 📝 Example Files

- `simple.js` - A minimal JavaScript file for testing