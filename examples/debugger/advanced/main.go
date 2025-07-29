package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	var breakpoints string
	flag.StringVar(&breakpoints, "bp", "", "Breakpoints in format [line1,line2,...] e.g., [5,10,15]")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-bp line1,line2,...] script.js\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s -bp 5,10,15 script.js\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "     or: %s -bp \"[5,10,15]\" script.js\n", os.Args[0])
		os.Exit(1)
	}

	filename := args[0]

	// Read the script file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse breakpoints
	var bpLines []int
	if breakpoints != "" {
		// Remove brackets (if present) and spaces
		bp := strings.Trim(breakpoints, "[]")
		bp = strings.ReplaceAll(bp, " ", "")
		
		if bp != "" {
			parts := strings.Split(bp, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				line, err := strconv.Atoi(part)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid breakpoint line: %s\n", part)
					continue
				}
				bpLines = append(bpLines, line)
			}
		}
	}

	// Create runtime and enable debugger
	vm := goja.New()
	debugger := vm.EnableDebugger()

	// Set up console.log
	console := vm.NewObject()
	console.Set("log", func(args ...interface{}) {
		fmt.Print("console.log: ")
		for i, arg := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(arg)
		}
		fmt.Println()
	})
	vm.Set("console", console)

	// Split source into lines for display
	sourceLines := strings.Split(string(content), "\n")

	// Track current state
	reader := bufio.NewReader(os.Stdin)
	lastLine := -1
	stepMode := false
	pauseCount := 0

	// Set up debug handler
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		line := state.SourcePos.Line
		
		// Skip if we're on the same line (multiple instructions per line)
		if line == lastLine && !stepMode {
			return goja.DebugContinue
		}
		lastLine = line

		pauseCount++
		
		// Clear screen (works on most terminals)
		fmt.Print("\033[H\033[2J")
		
		// Print header
		fmt.Printf("=== DEBUGGER PAUSED (pause #%d) ===\n", pauseCount)
		fmt.Printf("File: %s, Line: %d\n", filename, line)
		
		if state.Breakpoint != nil {
			fmt.Printf("Hit breakpoint at line %d\n", line)
		}
		
		fmt.Println("\n--- Source Code ---")
		
		// Show source code with context
		start := line - 3
		if start < 1 {
			start = 1
		}
		end := line + 3
		if end > len(sourceLines) {
			end = len(sourceLines)
		}
		
		for i := start; i <= end && i <= len(sourceLines); i++ {
			prefix := "   "
			if i == line {
				prefix = "=> " // Current line marker
			}
			
			// Check if this line has a breakpoint
			hasBP := false
			for _, bp := range bpLines {
				if bp == i {
					hasBP = true
					break
				}
			}
			
			if hasBP {
				fmt.Printf("%s%4d [BP]: %s\n", prefix, i, sourceLines[i-1])
			} else {
				fmt.Printf("%s%4d:     %s\n", prefix, i, sourceLines[i-1])
			}
		}
		
		// Show call stack
		fmt.Println("\n--- Call Stack ---")
		stack := vm.CaptureCallStack(10, nil)
		for i, frame := range stack {
			funcName := "(anonymous)"
			if frame.FuncName() != "" {
				funcName = frame.FuncName()
			}
			pos := frame.Position()
			fmt.Printf("  #%d: %s at %s:%d\n", i, funcName, pos.Filename, pos.Line)
		}
		
		// Show commands
		fmt.Println("\n--- Commands ---")
		fmt.Println("  [enter]    - Continue to next breakpoint")
		fmt.Println("  s [enter]  - Step to next line")
		fmt.Println("  o [enter]  - Step over function calls")
		fmt.Println("  u [enter]  - Step out of current function")
		fmt.Println("  c [enter]  - Continue execution (disable all breakpoints)")
		fmt.Println("  q [enter]  - Quit debugger")
		
		fmt.Print("\nCommand: ")
		
		// Read user input
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		
		switch input {
		case "q":
			fmt.Println("Exiting debugger...")
			os.Exit(0)
		case "s":
			stepMode = true
			return goja.DebugStepInto
		case "o":
			stepMode = true
			return goja.DebugStepOver
		case "u":
			stepMode = true
			return goja.DebugStepOut
		case "c":
			fmt.Println("Continuing execution...")
			stepMode = false
			debugger.SetStepMode(false)
			// Remove all breakpoints to continue without interruption
			for _, bp := range debugger.GetBreakpoints() {
				debugger.RemoveBreakpoint(bp.ID())
			}
			return goja.DebugContinue
		default:
			// Default: continue to next breakpoint
			stepMode = false
			return goja.DebugContinue
		}
		
		return goja.DebugContinue // Should never reach here
	})

	// Add breakpoints
	for _, line := range bpLines {
		id := debugger.AddBreakpoint(filename, line, 0)
		fmt.Printf("Added breakpoint at line %d (ID: %d)\n", line, id)
	}

	// If no breakpoints, enable step mode
	if len(bpLines) == 0 {
		fmt.Println("No breakpoints specified. Starting in step mode...")
		debugger.SetStepMode(true)
		stepMode = true
	}

	fmt.Println("\nPress [enter] to start debugging...")
	reader.ReadString('\n')

	// Run the script
	fmt.Println("Starting script execution...\n")
	result, err := vm.RunScript(filename, string(content))
	
	// Clear screen one more time
	fmt.Print("\033[H\033[2J")
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n=== SCRIPT ERROR ===\n")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("=== SCRIPT COMPLETED ===")
	if result != nil && !goja.IsUndefined(result) && !goja.IsNull(result) {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Printf("\nTotal pauses: %d\n", pauseCount)
}