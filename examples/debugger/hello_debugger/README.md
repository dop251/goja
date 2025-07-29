# Hello Debugger - Goja Debugger Quick Start

This is the simplest possible example of using the Goja debugger. Perfect for getting started!

## 🚀 Quick Start (30 seconds)

```bash
# 1. Build the example
go build -o hello_debugger main.go

# 2. Run it
./hello_debugger

# 3. Press ENTER to step through the code!
```

## What it does

This example runs a simple JavaScript program that:
1. Defines a greeting function
2. Calls the function with "World"
3. Prints the result

The debugger will pause at each line, showing you exactly what's happening.

## Building and Running

```bash
# Build the example
go build -o hello_debugger main.go

# Run it
./hello_debugger
```

## What you'll see

```
=== Goja Debugger Quick Start ===

This example will pause at each line of JavaScript code.
Press ENTER to step through the code.

Starting execution...

📍 Line 3: function greet(name) {
Press ENTER to continue...

📍 Line 9: var result = greet("World");
Press ENTER to continue...

📍 Line 4: var message = "Hello, " + name + "!";
Press ENTER to continue...

📍 Line 5: console.log(message);
Press ENTER to continue...
[JavaScript Output] Hello, World!

📍 Line 6: return message;
Press ENTER to continue...

📍 Line 10: console.log("Done!");
Press ENTER to continue...
[JavaScript Output] Done!

✅ Script completed successfully!
```

## How it works

1. **Create a Runtime**: `vm := goja.New()` creates a JavaScript runtime
2. **Enable Debugger**: `debugger := vm.EnableDebugger()` turns on debugging
3. **Set Handler**: The handler function is called each time the debugger pauses
4. **Enable Step Mode**: `debugger.SetStepMode(true)` makes it pause at each line
5. **Run Script**: `vm.RunString(script)` executes the JavaScript

## Key Concepts

- **Step Mode**: Pauses at every line of code
- **Debug Handler**: Your function that gets called when paused
- **Debug Commands**: Tell the debugger what to do next:
  - `DebugStepInto`: Go to the next line (including into functions)
  - `DebugStepOver`: Go to the next line (skip over functions)
  - `DebugContinue`: Run until the next breakpoint

## More Examples in This Directory

1. **`main.go`** - The basic example (you are here!)
2. **`example_with_breakpoints.go`** - Shows how to use breakpoints
3. **`run_file.go`** - Debug JavaScript files from disk

Try them all:
```bash
# Example with breakpoints
go build -o example_with_breakpoints example_with_breakpoints.go
./example_with_breakpoints

# Debug a file
go build -o run_file run_file.go
./run_file simple_debug.js
```

## Next Steps

Once you understand these examples, check out the more advanced debugger in `../advanced/` which shows:
- Command-line breakpoint specification
- Different stepping modes (step over, step out)
- Full call stack viewing
- More complex JavaScript programs