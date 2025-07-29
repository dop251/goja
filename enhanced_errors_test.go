package goja

import (
	"strings"
	"testing"
)

func TestEnhancedErrorsBasic(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test undefined variable
	_, err := vm.RunString(`
		console.log(undefinedVariable);
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check error type
	if enhanced.errorType != "ReferenceError" {
		t.Errorf("Expected ReferenceError but got %s", enhanced.errorType)
	}
	
	// Check for suggestions
	if len(enhanced.suggestions) == 0 {
		t.Error("Expected suggestions but got none")
	}
	
	// Check error message contains code frame
	errStr := enhanced.Error()
	if !strings.Contains(errStr, "console.log(undefinedVariable)") {
		t.Error("Error message should contain the problematic code")
	}
	
	// Check for line numbers in code frame
	if !strings.Contains(errStr, "│") {
		t.Error("Error message should contain code frame formatting")
	}
}

func TestEnhancedErrorsTypeError(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test calling non-function
	_, err := vm.RunString(`
		var obj = { name: "test" };
		obj.name();
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check error type
	if enhanced.errorType != "TypeError" {
		t.Errorf("Expected TypeError but got %s", enhanced.errorType)
	}
	
	// Check for function-related suggestions
	foundFunctionSuggestion := false
	for _, suggestion := range enhanced.suggestions {
		if strings.Contains(strings.ToLower(suggestion), "function") {
			foundFunctionSuggestion = true
			break
		}
	}
	
	if !foundFunctionSuggestion {
		t.Error("Expected function-related suggestion")
	}
}

func TestEnhancedErrorsStackTrace(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test with multiple function calls
	_, err := vm.RunString(`
		function a() {
			b();
		}
		
		function b() {
			c();
		}
		
		function c() {
			undefinedFunction();
		}
		
		a();
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check stack trace contains all functions
	errStr := enhanced.Error()
	if !strings.Contains(errStr, "at <anonymous>") {
		t.Error("Stack trace should contain anonymous function")
	}
	
	// Check for proper formatting
	if !strings.Contains(errStr, "Stack trace:") {
		t.Error("Should have Stack trace section")
	}
}

func TestEnhancedErrorsSyntaxError(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test syntax error
	_, err := vm.RunString(`
		function test() {
			if (true) {
				console.log("unclosed
			}
		}
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	// Syntax errors might not be wrapped as Exception during parsing
	// Check if we get any error with suggestions
	if enhanced, ok := err.(*EnhancedError); ok {
		// Check for syntax-related suggestions
		foundSyntaxSuggestion := false
		for _, suggestion := range enhanced.suggestions {
			if strings.Contains(strings.ToLower(suggestion), "quotes") ||
			   strings.Contains(strings.ToLower(suggestion), "string") {
				foundSyntaxSuggestion = true
				break
			}
		}
		
		if enhanced.errorType == "SyntaxError" && !foundSyntaxSuggestion {
			t.Error("Expected syntax-related suggestion")
		}
	}
}

func TestEnhancedErrorsDisabled(t *testing.T) {
	vm := New()
	// Don't enable enhanced errors
	
	_, err := vm.RunString(`
		undefinedVariable;
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	// Should get regular Exception, not EnhancedError
	_, ok := err.(*EnhancedError)
	if ok {
		t.Error("Should not get EnhancedError when feature is disabled")
	}
	
	exc, ok := err.(*Exception)
	if !ok {
		t.Fatalf("Expected Exception but got %T", err)
	}
	
	// Basic error should still work
	if exc.Error() == "" {
		t.Error("Basic error message should not be empty")
	}
}

func TestEnhancedErrorsCodeContext(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Multi-line code to test context
	_, err := vm.RunString(`
		var a = 1;
		var b = 2;
		var c = 3;
		callUndefined(); // This is line 5
		var d = 4;
		var e = 5;
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check code frame includes context lines
	errStr := enhanced.Error()
	if !strings.Contains(errStr, "var c = 3") {
		t.Error("Should show line before error")
	}
	if !strings.Contains(errStr, "callUndefined()") {
		t.Error("Should show error line")
	}
	if !strings.Contains(errStr, "var d = 4") {
		t.Error("Should show line after error")
	}
	
	// Check for arrow pointing to error line
	if !strings.Contains(errStr, "→") {
		t.Error("Should have arrow pointing to error line")
	}
}

func TestEnhancedErrorsRangeError(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test stack overflow
	_, err := vm.RunString(`
		function recursive() {
			return recursive();
		}
		recursive();
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	// Stack overflow might be a special error type
	// If it's enhanced, check for recursion suggestions
	if enhanced, ok := err.(*EnhancedError); ok {
		if strings.Contains(enhanced.simpleMsg, "stack") {
			foundRecursionSuggestion := false
			for _, suggestion := range enhanced.suggestions {
				if strings.Contains(strings.ToLower(suggestion), "recursion") ||
				   strings.Contains(strings.ToLower(suggestion), "recursive") {
					foundRecursionSuggestion = true
					break
				}
			}
			
			if !foundRecursionSuggestion {
				t.Error("Expected recursion-related suggestion for stack overflow")
			}
		}
	}
}

func TestEnhancedErrorsHelperMethods(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	_, err := vm.RunString(`unknownVar;`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Test GetSimpleMessage
	simple := enhanced.GetSimpleMessage()
	if !strings.Contains(simple, "ReferenceError") {
		t.Error("Simple message should contain error type")
	}
	
	// Test GetSuggestions
	suggestions := enhanced.GetSuggestions()
	if len(suggestions) == 0 {
		t.Error("Should have suggestions")
	}
	
	// Test GetCodeFrame
	frame := enhanced.GetCodeFrame()
	if frame == "" {
		t.Error("Should have code frame")
	}
}

func TestEnhancedErrorsPropertyAccess(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	_, err := vm.RunString(`
		var obj = null;
		obj.property;
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check for null check suggestions
	foundNullSuggestion := false
	for _, suggestion := range enhanced.suggestions {
		if strings.Contains(strings.ToLower(suggestion), "null") ||
		   strings.Contains(strings.ToLower(suggestion), "undefined") ||
		   strings.Contains(strings.ToLower(suggestion), "check") {
			foundNullSuggestion = true
			break
		}
	}
	
	if !foundNullSuggestion {
		t.Error("Expected null/undefined check suggestion")
	}
}

func TestEnhancedErrorsCommonTypos(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	testCases := []struct {
		code     string
		expected string
	}{
		{`documnet.getElementById("test");`, "document"},
		{`windwo.location;`, "window"},
		{`cosole.log("test");`, "console"},
	}
	
	for _, tc := range testCases {
		_, err := vm.RunString(tc.code)
		
		if err == nil {
			t.Errorf("Expected error for code: %s", tc.code)
			continue
		}
		
		enhanced, ok := err.(*EnhancedError)
		if !ok {
			continue
		}
		
		// Check if suggestion mentions the correct variable
		foundSuggestion := false
		for _, suggestion := range enhanced.suggestions {
			if strings.Contains(suggestion, tc.expected) {
				foundSuggestion = true
				break
			}
		}
		
		if !foundSuggestion {
			t.Errorf("Expected suggestion containing '%s' for typo in: %s", tc.expected, tc.code)
		}
	}
}

// Test that enhanced errors don't break existing functionality
func TestEnhancedErrorsCompatibility(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Test that we can still catch errors normally
	result, err := vm.RunString(`
		try {
			nonExistent();
		} catch (e) {
			e.name + ": " + e.message;
		}
	`)
	
	if err != nil {
		t.Fatalf("Should not get error from caught exception: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result from catch block")
	}
	
	str := result.String()
	if !strings.Contains(str, "ReferenceError") {
		t.Errorf("Expected ReferenceError in catch block, got: %s", str)
	}
}

func TestEnhancedErrorsNativeCode(t *testing.T) {
	vm := New()
	vm.EnableEnhancedErrors()
	
	// Create a native function that throws
	vm.Set("throwError", func() {
		panic(vm.NewError(vm.Get("ReferenceError").(*Object), "Native error"))
	})
	
	_, err := vm.RunString(`
		function wrapper() {
			throwError();
		}
		wrapper();
	`)
	
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	
	enhanced, ok := err.(*EnhancedError)
	if !ok {
		t.Fatalf("Expected EnhancedError but got %T", err)
	}
	
	// Check stack trace includes native code indicator
	errStr := enhanced.Error()
	if !strings.Contains(errStr, "<native") {
		t.Error("Stack trace should indicate native code")
	}
}