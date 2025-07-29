# Enhanced Errors Example

This example demonstrates Goja's enhanced error messages feature that provides:

- **Code context** with line numbers
- **Helpful suggestions** based on error type
- **Improved stack traces** with code snippets
- **Better formatting** for easier debugging

## Features

### 1. Code Context
Shows the code around where the error occurred with line numbers and an arrow pointing to the exact location.

```
ReferenceError: priceTax is not defined

  3 │ function calculateTotal() {
  4 │     var price = 100;
  5 │     var tax = 0.08;
→ 6 │     return price + priceTax;  // Typo: should be 'price * tax'
    │                    ^
  7 │ }
```

### 2. Intelligent Suggestions
Provides context-aware suggestions based on the error:

```
💡 Suggestions:
   1. JavaScript is case-sensitive. Did you mean 'pricetax'?
   2. Check if 'priceTax' is spelled correctly
   3. Make sure 'priceTax' is defined before using it
   4. If it's a global variable, check if the script/library is loaded
```

### 3. Enhanced Stack Traces
Shows the call stack with code snippets:

```
Stack trace:
  1. calculateTotal at example.js:6:20
     | return price + priceTax;
  2. <anonymous> at example.js:9:1
     | calculateTotal();
```

## Running the Example

```bash
go run main.go
```

## Enabling Enhanced Errors

To use enhanced errors in your code:

```go
vm := goja.New()
vm.EnableEnhancedErrors()

// Now all errors will be enhanced
_, err := vm.RunString(code)
if err != nil {
    fmt.Println(err) // Will show enhanced error
}
```

## Accessing Error Components

Enhanced errors provide methods to access individual components:

```go
if enhanced, ok := err.(*goja.EnhancedError); ok {
    // Get just the error message
    simple := enhanced.GetSimpleMessage()
    
    // Get suggestions array
    suggestions := enhanced.GetSuggestions()
    
    // Get code frame
    frame := enhanced.GetCodeFrame()
}
```

## Error Types Supported

The enhanced errors provide specialized suggestions for:

- **ReferenceError**: Undefined variables, typos
- **TypeError**: Invalid operations, null/undefined access
- **RangeError**: Stack overflow, array bounds
- **SyntaxError**: Missing brackets, unclosed strings

## Benefits

1. **Faster Debugging**: See exactly where and why errors occur
2. **Learning Tool**: Suggestions help understand JavaScript better
3. **Typo Detection**: Common misspellings are detected
4. **Context Awareness**: Shows surrounding code for better understanding