package goja

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// EnhancedError provides improved error messages with code context and suggestions
type EnhancedError struct {
	*Exception
	codeFrame   string
	suggestions []string
	errorType   string
	simpleMsg   string
}

// CodeContext shows the code around the error with line numbers
type CodeContext struct {
	Lines      []string
	ErrorLine  int
	ErrorCol   int
	StartLine  int
}

// NewEnhancedError wraps an Exception with additional context
func NewEnhancedError(e *Exception) *EnhancedError {
	enhanced := &EnhancedError{
		Exception: e,
	}
	
	// Analyze the error and add enhancements
	enhanced.analyze()
	enhanced.buildCodeFrame()
	enhanced.generateSuggestions()
	
	return enhanced
}

// Error returns the enhanced error message
func (e *EnhancedError) Error() string {
	var buf bytes.Buffer
	
	// Error type and message
	buf.WriteString(fmt.Sprintf("%s: %s\n", e.errorType, e.simpleMsg))
	
	// Code frame if available
	if e.codeFrame != "" {
		buf.WriteString("\n")
		buf.WriteString(e.codeFrame)
		buf.WriteString("\n")
	}
	
	// Suggestions
	if len(e.suggestions) > 0 {
		buf.WriteString("\n💡 Suggestions:\n")
		for i, suggestion := range e.suggestions {
			buf.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
		}
	}
	
	// Stack trace
	buf.WriteString("\nStack trace:\n")
	e.writeEnhancedStack(&buf)
	
	return buf.String()
}

// analyze determines the error type and simple message
func (e *EnhancedError) analyze() {
	if e.val == nil {
		e.errorType = "Error"
		e.simpleMsg = "Unknown error"
		return
	}
	
	// Try to get error type from object
	if obj, ok := e.val.(*Object); ok {
		if nameVal := obj.self.getStr("name", nil); nameVal != nil {
			e.errorType = nameVal.String()
		} else {
			e.errorType = "Error"
		}
		
		if msgVal := obj.self.getStr("message", nil); msgVal != nil {
			e.simpleMsg = msgVal.String()
		} else {
			e.simpleMsg = e.val.String()
		}
	} else {
		e.errorType = "Error"
		e.simpleMsg = e.val.String()
	}
}

// buildCodeFrame creates a visual representation of where the error occurred
func (e *EnhancedError) buildCodeFrame() {
	if len(e.stack) == 0 {
		return
	}
	
	frame := e.stack[0]
	if frame.prg == nil || frame.prg.src == nil {
		return
	}
	
	ctx := e.getCodeContext(frame)
	if ctx == nil {
		return
	}
	
	var buf bytes.Buffer
	
	// Build the code frame
	maxLineNumWidth := len(fmt.Sprintf("%d", ctx.StartLine+len(ctx.Lines)))
	
	for i, line := range ctx.Lines {
		lineNum := ctx.StartLine + i
		isErrorLine := lineNum == ctx.ErrorLine
		
		// Line number
		lineNumStr := fmt.Sprintf("%*d", maxLineNumWidth, lineNum)
		
		if isErrorLine {
			buf.WriteString(fmt.Sprintf("→ %s │ %s\n", lineNumStr, line))
			
			// Error pointer
			if ctx.ErrorCol > 0 && ctx.ErrorCol <= len(line) {
				buf.WriteString(fmt.Sprintf("  %s │ ", strings.Repeat(" ", maxLineNumWidth)))
				buf.WriteString(strings.Repeat(" ", ctx.ErrorCol-1))
				buf.WriteString("^\n")
			}
		} else {
			buf.WriteString(fmt.Sprintf("  %s │ %s\n", lineNumStr, line))
		}
	}
	
	// Add file info
	buf.WriteString(fmt.Sprintf("\n  at %s:%d:%d", frame.SrcName(), ctx.ErrorLine, ctx.ErrorCol))
	
	e.codeFrame = buf.String()
}

// getCodeContext extracts lines around the error
func (e *EnhancedError) getCodeContext(frame StackFrame) *CodeContext {
	if frame.prg == nil || frame.prg.src == nil {
		return nil
	}
	
	// Get position info
	pos := frame.Position()
	if pos.Line <= 0 {
		return nil
	}
	
	// Get source code
	src := frame.prg.src.Source()
	if src == "" {
		return nil
	}
	
	lines := strings.Split(src, "\n")
	if pos.Line > len(lines) {
		return nil
	}
	
	// Calculate context lines (3 before, 3 after)
	startLine := pos.Line - 3
	if startLine < 1 {
		startLine = 1
	}
	
	endLine := pos.Line + 3
	if endLine > len(lines) {
		endLine = len(lines)
	}
	
	contextLines := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		if i-1 < len(lines) {
			contextLines = append(contextLines, lines[i-1])
		}
	}
	
	return &CodeContext{
		Lines:     contextLines,
		ErrorLine: pos.Line,
		ErrorCol:  pos.Column,
		StartLine: startLine,
	}
}

// generateSuggestions creates helpful suggestions based on the error
func (e *EnhancedError) generateSuggestions() {
	e.suggestions = []string{}
	
	msg := strings.ToLower(e.simpleMsg)
	errorType := strings.ToLower(e.errorType)
	
	// ReferenceError suggestions
	if errorType == "referenceerror" {
		if strings.Contains(msg, "is not defined") {
			// Extract variable name
			parts := strings.Split(e.simpleMsg, " ")
			if len(parts) > 0 {
				varName := parts[0]
				e.addVariableSuggestions(varName)
			}
		}
	}
	
	// TypeError suggestions
	if errorType == "typeerror" {
		if strings.Contains(msg, "is not a function") {
			e.suggestions = append(e.suggestions, 
				"Check if the variable is defined before calling it",
				"Verify that you're calling the correct method name",
				"Make sure the object has the method you're trying to call")
		} else if strings.Contains(msg, "cannot read property") {
			e.suggestions = append(e.suggestions,
				"Check if the object exists before accessing its properties",
				"Use optional chaining (?.) to safely access nested properties",
				"Add a null/undefined check before property access")
		} else if strings.Contains(msg, "undefined is not an object") {
			e.suggestions = append(e.suggestions,
				"Initialize the variable before using it",
				"Check if the function returns a value",
				"Verify that async operations have completed")
		}
	}
	
	// SyntaxError suggestions
	if errorType == "syntaxerror" {
		if strings.Contains(msg, "unexpected token") {
			e.suggestions = append(e.suggestions,
				"Check for missing semicolons or commas",
				"Verify parentheses, brackets, and braces are balanced",
				"Look for typos in keywords (e.g., 'funtion' instead of 'function')")
		} else if strings.Contains(msg, "unexpected end") {
			e.suggestions = append(e.suggestions,
				"Check if all opened brackets/braces are closed",
				"Verify that all string quotes are closed",
				"Look for incomplete statements")
		}
	}
	
	// RangeError suggestions
	if errorType == "rangeerror" {
		if strings.Contains(msg, "maximum call stack") {
			e.suggestions = append(e.suggestions,
				"Check for infinite recursion in your functions",
				"Add a base case to recursive functions",
				"Consider using iteration instead of recursion")
		}
	}
}

// addVariableSuggestions adds suggestions for undefined variable errors
func (e *EnhancedError) addVariableSuggestions(varName string) {
	// Check for common typos
	suggestions := []string{}
	
	// Common typos
	if varName == "documnet" {
		suggestions = append(suggestions, "Did you mean 'document'?")
	} else if varName == "windwo" || varName == "widnow" {
		suggestions = append(suggestions, "Did you mean 'window'?")
	} else if varName == "cosole" || varName == "consol" {
		suggestions = append(suggestions, "Did you mean 'console'?")
	}
	
	// Check case sensitivity
	if varName != "" && unicode.IsUpper(rune(varName[0])) {
		suggestions = append(suggestions, fmt.Sprintf("JavaScript is case-sensitive. Did you mean '%s'?", 
			strings.ToLower(string(varName[0]))+varName[1:]))
	}
	
	// General suggestions
	suggestions = append(suggestions,
		fmt.Sprintf("Check if '%s' is spelled correctly", varName),
		fmt.Sprintf("Make sure '%s' is defined before using it", varName),
		"If it's a global variable, check if the script/library is loaded")
	
	e.suggestions = suggestions
}

// writeEnhancedStack writes an improved stack trace
func (e *EnhancedError) writeEnhancedStack(buf *bytes.Buffer) {
	for i, frame := range e.stack {
		// Frame number
		buf.WriteString(fmt.Sprintf("  %d. ", i+1))
		
		// Function name or anonymous
		if frame.funcName != "" {
			buf.WriteString(frame.funcName.String())
		} else {
			buf.WriteString("<anonymous>")
		}
		
		// Location
		buf.WriteString(" at ")
		if frame.prg != nil {
			pos := frame.Position()
			buf.WriteString(fmt.Sprintf("%s:%d:%d", frame.SrcName(), pos.Line, pos.Column))
		} else {
			buf.WriteString("<native code>")
		}
		
		buf.WriteString("\n")
		
		// Add code snippet for top frames
		if i < 3 && frame.prg != nil {
			if snippet := e.getLineSnippet(frame); snippet != "" {
				buf.WriteString(fmt.Sprintf("     %s\n", snippet))
			}
		}
	}
}

// getLineSnippet returns a single line of code for the frame
func (e *EnhancedError) getLineSnippet(frame StackFrame) string {
	if frame.prg == nil || frame.prg.src == nil {
		return ""
	}
	
	pos := frame.Position()
	if pos.Line <= 0 {
		return ""
	}
	
	src := frame.prg.src.Source()
	lines := strings.Split(src, "\n")
	
	if pos.Line <= len(lines) {
		line := strings.TrimSpace(lines[pos.Line-1])
		if len(line) > 60 {
			line = line[:57] + "..."
		}
		return fmt.Sprintf("| %s", line)
	}
	
	return ""
}

// GetSimpleMessage returns just the error message without enhancements
func (e *EnhancedError) GetSimpleMessage() string {
	return fmt.Sprintf("%s: %s", e.errorType, e.simpleMsg)
}

// GetSuggestions returns the suggestions array
func (e *EnhancedError) GetSuggestions() []string {
	return e.suggestions
}

// GetCodeFrame returns the code frame string
func (e *EnhancedError) GetCodeFrame() string {
	return e.codeFrame
}