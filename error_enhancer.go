package goja

import "fmt"

// ErrorEnhancer is an interface for customizing error messages
type ErrorEnhancer interface {
	EnhanceError(err error) error
}

// DefaultErrorEnhancer provides the default error enhancement
type DefaultErrorEnhancer struct {
	enabled bool
}

// EnhanceError enhances an error if it's an Exception
func (d *DefaultErrorEnhancer) EnhanceError(err error) error {
	if !d.enabled {
		return err
	}
	
	if exc, ok := err.(*Exception); ok {
		return NewEnhancedError(exc)
	}
	
	return err
}

// EnableEnhancedErrors enables enhanced error messages for this runtime
func (r *Runtime) EnableEnhancedErrors() {
	r.enhancedErrors = true
}

// DisableEnhancedErrors disables enhanced error messages for this runtime
func (r *Runtime) DisableEnhancedErrors() {
	r.enhancedErrors = false
}

// enhanceError enhances an error if enhanced errors are enabled
func (r *Runtime) enhanceError(err error) error {
	if !r.enhancedErrors {
		return err
	}
	
	if exc, ok := err.(*Exception); ok {
		return NewEnhancedError(exc)
	}
	
	return err
}

// NewError creates a new enhanced error if enabled
func (r *Runtime) NewError(typ *Object, format string, args ...interface{}) error {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	
	exc := &Exception{
		val:   r.builtin_new(typ, []Value{newStringValue(msg)}),
		stack: r.CaptureCallStack(0, nil),
	}
	
	if r.enhancedErrors {
		return NewEnhancedError(exc)
	}
	
	return exc
}

// WrapError wraps a Go error as a JavaScript error
func (r *Runtime) WrapError(err error) error {
	if err == nil {
		return nil
	}
	
	// If it's already an Exception, just enhance it if needed
	if exc, ok := err.(*Exception); ok {
		if r.enhancedErrors {
			return NewEnhancedError(exc)
		}
		return exc
	}
	
	// Create a new GoError
	exc := &Exception{
		val:   r.builtin_new(r.getGoError(), []Value{r.ToValue(err)}),
		stack: r.CaptureCallStack(0, nil),
	}
	
	if r.enhancedErrors {
		return NewEnhancedError(exc)
	}
	
	return exc
}