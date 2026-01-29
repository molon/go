// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"runtime"
)

// StackFrame represents a program counter inside a stack frame.
// For historical reasons if StackFrame is interpreted as a uintptr
// its value represents the program counter + 1.
type StackFrame uintptr

// PC returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f StackFrame) PC() uintptr { return uintptr(f) - 1 }

// StackTrace is stack of StackFrames from innermost (newest) to outermost (oldest).
type StackTrace []StackFrame

// Format formats the stack trace using FormatStackTrace.
func (st StackTrace) Format(write func(s string)) {
	FormatStackTrace(st, write)
}

// FormatStackTrace is the default stack trace formatter.
// It can be replaced with a custom implementation.
// Each frame is formatted as:
//
//	function.name
//		/path/to/file.go:line
var FormatStackTrace = func(st StackTrace, write func(s string)) {
	for _, f := range st {
		pc := f.PC()
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			write("\nunknown")
			continue
		}
		file, line := fn.FileLine(pc)
		write("\n")
		write(fn.Name())
		write("\n\t")
		write(file)
		write(":")
		write(itoa(line))
	}
}

// StackTracer is implemented by errors that have a stack trace.
type StackTracer interface {
	StackTrace() StackTrace
	HasStack() bool
}

// Stack represents a captured stack trace that can be embedded in error types.
//
// When Stack is embedded as an anonymous field (e.g., `*Stack` without a field name),
// [reflect.DeepEqual] will skip comparing it. This ensures that errors with identical
// messages but different stack traces are considered equal by DeepEqual.
// Named fields of type *Stack are still compared normally.
type Stack struct {
	pcs []uintptr
}

// CaptureStack captures the current stack trace, skipping the specified number of frames.
// skip=0 means start from the caller of CaptureStack.
func CaptureStack(skip int) *Stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	return &Stack{pcs: pcs[0:n:n]}
}

// StackTrace returns the stack trace as a slice of StackFrames.
func (s *Stack) StackTrace() StackTrace {
	if s == nil || s.pcs == nil {
		return nil
	}
	f := make([]StackFrame, len(s.pcs))
	for i := 0; i < len(f); i++ {
		f[i] = StackFrame(s.pcs[i])
	}
	return f
}

// HasStack returns true if this Stack contains a captured stack trace.
func (s *Stack) HasStack() bool {
	return s != nil && len(s.pcs) > 0
}

// RemoveStackFrames removes the first n frames from the stack trace.
// This is used internally to adjust the stack when errors are created through wrapper functions.
func (s *Stack) RemoveStackFrames(n int) {
	if s != nil && n > 0 && n < len(s.pcs) {
		s.pcs = s.pcs[n:]
	}
}

// ShouldCaptureStack checks if an error needs a stack trace to be captured.
// Returns true if err is nil or any branch in the error tree is missing a stack trace.
// This is a variable so it can be replaced by custom logic if needed.
var ShouldCaptureStack = func(err error) bool {
	return needsCaptureStackImpl(err, 0)
}

func needsCaptureStackImpl(err error, depth int) bool {
	// nil error needs stack capture
	if err == nil {
		return true
	}
	// Prevent infinite loops, assume needs stack if too deep
	if depth > 100 {
		return true
	}

	// If this error has a stack trace (and actually has one), this branch is covered
	if st, ok := err.(StackTracer); ok && st.HasStack() {
		return false
	}

	// Check for Unwrap() []error first (multi-error)
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		errs := u.Unwrap()
		if len(errs) == 0 {
			return true
		}
		for _, e := range errs {
			if needsCaptureStackImpl(e, depth+1) {
				return true
			}
		}
		return false
	}

	// Check for Unwrap() error (single-error)
	if u, ok := err.(interface{ Unwrap() error }); ok {
		inner := u.Unwrap()
		if inner == nil {
			return true
		}
		return needsCaptureStackImpl(inner, depth+1)
	}

	// Leaf node without stack trace
	return true
}

// itoa converts an integer to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}
