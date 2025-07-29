# Advanced Debugger Example

This is a full-featured debugger for Goja with command-line interface, breakpoints, and advanced stepping capabilities.

## Building

```bash
go build -o debugger main.go
```

## Usage

```bash
./debugger [-bp [line1,line2,...]] script.js
```

### Examples

1. **Run with breakpoints at specific lines:**
   ```bash
   # Option 1: Use quotes
   ./debugger -bp "[5,10,15]" test1.js
   
   # Option 2: Escape brackets (for zsh)
   ./debugger -bp \[5,10,15\] test1.js
   
   # Option 3: Use simple comma-separated format
   ./debugger -bp 5,10,15 test1.js
   ```

2. **Run in step mode (no breakpoints):**
   ```bash
   ./debugger test1.js
   ```

3. **Debug the Fibonacci example with breakpoints:**
   ```bash
   ./debugger -bp "[3,11,18]" test2.js
   ```

## Commands

When the debugger pauses, you can use these commands:

- **[enter]** - Continue to next breakpoint
- **s [enter]** - Step to next line (step into functions)
- **o [enter]** - Step over function calls
- **u [enter]** - Step out of current function
- **c [enter]** - Continue execution (disable all breakpoints)
- **q [enter]** - Quit debugger

## Features

- Visual source code display with current line marker
- Breakpoint indicators [BP]
- Call stack display
- Console.log support
- Clear command interface

## Example Session

```
=== DEBUGGER PAUSED (pause #1) ===
File: test1.js, Line: 5

Hit breakpoint at line 5

--- Source Code ---
      2:     var x = 10;
      3:     var y = 20;
      4:     var sum = x + y;
=>    5 [BP]: console.log("The sum is:", sum);
      6:     
      7:     // Function example
      8:     function multiply(a, b) {

--- Call Stack ---
  #0: (anonymous) at test1.js:5

--- Commands ---
  [enter]    - Continue to next breakpoint
  s [enter]  - Step to next line
  o [enter]  - Step over function calls
  u [enter]  - Step out of current function
  c [enter]  - Continue execution (disable all breakpoints)
  q [enter]  - Quit debugger

Command: 
```