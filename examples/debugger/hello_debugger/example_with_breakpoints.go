// This example shows how to use breakpoints instead of stepping through every line
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	// JavaScript code with a loop
	script := `
function calculateSum(n) {
    var sum = 0;
    for (var i = 1; i <= n; i++) {
        sum = sum + i;  // Line 5 - we'll put a breakpoint here
    }
    return sum;
}

var result = calculateSum(5);
console.log("Sum of 1 to 5 is: " + result);
`

	fmt.Println("=== Breakpoint Example ===")
	fmt.Println("\nThis example shows how to use breakpoints.")
	fmt.Println("We'll set a breakpoint inside the loop at line 5.\n")

	// Create runtime and enable debugger
	vm := goja.New()
	debugger := vm.EnableDebugger()
	
	// Set up console.log
	console := vm.NewObject()
	console.Set("log", func(msg string) {
		fmt.Printf("[Output] %s\n", msg)
	})
	vm.Set("console", console)

	// Split script into lines
	lines := strings.Split(strings.TrimSpace(script), "\n")
	reader := bufio.NewReader(os.Stdin)
	
	pauseCount := 0
	
	// Debug handler
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		pauseCount++
		
		line := state.SourcePos.Line
		if line > 0 && line <= len(lines) {
			fmt.Printf("\n🔴 Breakpoint hit! (pause #%d)\n", pauseCount)
			fmt.Printf("Line %d: %s\n", line, strings.TrimSpace(lines[line-1]))
			
			// Show some context
			if state.Breakpoint != nil {
				fmt.Printf("This is breakpoint ID: %d\n", state.Breakpoint.ID())
			}
			
			fmt.Print("\nPress ENTER to continue...")
			reader.ReadString('\n')
		}
		
		return goja.DebugContinue
	})
	
	// Add a breakpoint at line 5 (inside the loop)
	bpID := debugger.AddBreakpoint("", 5, 0)
	fmt.Printf("Added breakpoint at line 5 (ID: %d)\n", bpID)
	
	fmt.Println("\nRunning script...")
	fmt.Println("The breakpoint will be hit 5 times (once for each loop iteration).\n")
	
	// Run the script
	_, err := vm.RunString(script)
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("\n✅ Done! The breakpoint was hit %d times.\n", pauseCount)
}

// Build and run with:
// go build -o example_with_breakpoints example_with_breakpoints.go
// ./example_with_breakpoints