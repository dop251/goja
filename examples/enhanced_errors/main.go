// This example demonstrates the enhanced error messages feature in Goja
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
)

func main() {
	fmt.Println("=== Goja Enhanced Errors Demo ===\n")

	examples := []struct {
		name string
		code string
	}{
		{
			name: "Undefined Variable",
			code: `
// Trying to use an undefined variable
function calculateTotal() {
    var price = 100;
    var tax = 0.08;
    return price + priceTax;  // Typo: should be 'price * tax'
}

calculateTotal();
`,
		},
		{
			name: "Type Error - Not a Function",
			code: `
var user = {
    name: "John",
    age: 30
};

// Trying to call a property as a function
user.name();
`,
		},
		{
			name: "Null Property Access",
			code: `
function getUser() {
    // Forgot to return a value
}

var user = getUser();
console.log(user.name);  // user is undefined
`,
		},
		{
			name: "Stack Overflow",
			code: `
function fibonacci(n) {
    // Missing base case!
    return fibonacci(n-1) + fibonacci(n-2);
}

fibonacci(10);
`,
		},
		{
			name: "Common Typo",
			code: `
// Common typo: 'document' misspelled
documnet.getElementById("myDiv");
`,
		},
	}

	// First, show regular errors
	fmt.Println("1. Regular Errors (Enhanced Errors Disabled)")
	fmt.Println(strings.Repeat("-", 60))
	
	vm := goja.New()
	_, err := vm.RunString(examples[0].code)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	
	fmt.Println("\n\n2. Enhanced Errors (Enabled)")
	fmt.Println(strings.Repeat("-", 60))
	
	// Now show enhanced errors
	for _, example := range examples {
		fmt.Printf("\n### %s ###\n", example.name)
		
		vm := goja.New()
		vm.EnableEnhancedErrors()
		
		// Set up console.log
		console := vm.NewObject()
		console.Set("log", func(msg string) {
			fmt.Printf("[console.log] %s\n", msg)
		})
		vm.Set("console", console)
		
		_, err := vm.RunString(example.code)
		if err != nil {
			fmt.Println(err)
			
			// If it's an enhanced error, we can also access components separately
			if enhanced, ok := err.(*goja.EnhancedError); ok {
				fmt.Println("\n[Components accessible separately:]")
				fmt.Printf("- Simple Message: %s\n", enhanced.GetSimpleMessage())
				fmt.Printf("- Suggestions: %d available\n", len(enhanced.GetSuggestions()))
			}
		}
		
		fmt.Println(strings.Repeat("-", 60))
	}
	
	fmt.Println("\n3. Programmatic Usage")
	fmt.Println(strings.Repeat("-", 60))
	demonstrateProgrammaticUsage()
}

func demonstrateProgrammaticUsage() {
	vm := goja.New()
	vm.EnableEnhancedErrors()
	
	_, err := vm.RunString(`
		var obj = { x: 10 };
		obj.y.z;  // Accessing nested property on undefined
	`)
	
	if err != nil {
		if enhanced, ok := err.(*goja.EnhancedError); ok {
			fmt.Println("Error components:")
			fmt.Printf("- Type: %s\n", enhanced.GetSimpleMessage())
			
			fmt.Println("\nSuggestions:")
			for i, suggestion := range enhanced.GetSuggestions() {
				fmt.Printf("  %d. %s\n", i+1, suggestion)
			}
			
			fmt.Println("\nCode Frame:")
			fmt.Println(enhanced.GetCodeFrame())
		}
	}
}