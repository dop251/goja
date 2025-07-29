// This is a simple example showing how to use the Goja debugger
// to debug JavaScript code with breakpoints and step-by-step execution.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	// The JavaScript code we want to debug
	script := `
// A simple greeting function
function greet(name) {
    var message = "Hello, " + name + "!";
    console.log(message);
    return message;
}

// Call the function
var result = greet("World");
console.log("Done!");
`

	fmt.Println("=== Goja Debugger Quick Start ===")
	fmt.Println("\nThis example will pause at each line of JavaScript code.")
	fmt.Println("Press ENTER to step through the code.\n")

	// Create a new JavaScript runtime
	vm := goja.New()
	
	// Enable the debugger
	debugger := vm.EnableDebugger()
	
	// Set up console.log so we can see output
	console := vm.NewObject()
	console.Set("log", func(msg string) {
		fmt.Printf("[JavaScript Output] %s\n", msg)
	})
	vm.Set("console", console)

	// Split the script into lines for display
	lines := strings.Split(strings.TrimSpace(script), "\n")
	
	// Create a reader for user input
	reader := bufio.NewReader(os.Stdin)
	
	// Keep track of the last line we showed
	lastLine := -1
	
	// Set up the debug handler - this function is called when the debugger pauses
	debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
		currentLine := state.SourcePos.Line
		
		// Don't show the same line twice
		if currentLine == lastLine || currentLine == 0 {
			return goja.DebugStepInto
		}
		lastLine = currentLine
		
		// Show the current line
		if currentLine > 0 && currentLine <= len(lines) {
			fmt.Printf("\n📍 Line %d: %s\n", currentLine, strings.TrimSpace(lines[currentLine-1]))
			fmt.Print("Press ENTER to continue...")
			reader.ReadString('\n')
		}
		
		// Continue to the next line
		return goja.DebugStepInto
	})
	
	// Enable step mode - this makes the debugger pause at each line
	debugger.SetStepMode(true)
	
	fmt.Println("Starting execution...\n")
	
	// Run the script
	_, err := vm.RunString(script)
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return
	}
	
	fmt.Println("\n✅ Script completed successfully!")
}