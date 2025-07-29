// This example shows how to debug a JavaScript file from disk
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./run_file <javascript_file>")
		fmt.Println("Example: ./run_file simple_debug.js")
		return
	}

	filename := os.Args[1]
	
	// Read the JavaScript file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("=== Debugging %s ===\n", filename)
	fmt.Println("\nPress ENTER to step through each line.\n")

	// Create runtime and enable debugger
	vm := goja.New()
	debugger := vm.EnableDebugger()
	
	// Set up console.log
	console := vm.NewObject()
	console.Set("log", func(args ...interface{}) {
		fmt.Print("[Output] ")
		for i, arg := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(arg)
		}
		fmt.Println()
	})
	vm.Set("console", console)

	// Split code into lines
	lines := strings.Split(string(content), "\n")
	reader := bufio.NewReader(os.Stdin)
	lastLine := -1
	
	// Debug handler
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		line := state.SourcePos.Line
		
		if line == lastLine || line == 0 {
			return goja.DebugStepInto
		}
		lastLine = line
		
		if line > 0 && line <= len(lines) {
			fmt.Printf("\n→ Line %d: %s\n", line, strings.TrimSpace(lines[line-1]))
			fmt.Print("  Press ENTER...")
			reader.ReadString('\n')
		}
		
		return goja.DebugStepInto
	})
	
	// Enable step mode
	debugger.SetStepMode(true)
	
	// Run the script
	_, err = vm.RunScript(filename, string(content))
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return
	}
	
	fmt.Println("\n✅ Done!")
}

// Build with: go build -o run_file run_file.go
// Run with: ./run_file simple_debug.js