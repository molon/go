// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt

import (
	"errors"
	"slices"
)

// Errorf formats according to a format specifier and returns the string as a
// value that satisfies error.
//
// If the format specifier includes a %w verb with an error operand,
// the returned error will implement an Unwrap method returning the operand.
// If there is more than one %w verb, the returned error will implement an
// Unwrap method returning a []error containing all the %w operands in the
// order they appear in the arguments.
// It is invalid to supply the %w verb with an operand that does not implement
// the error interface. The %w verb is otherwise a synonym for %v.
func Errorf(format string, a ...any) error {
	p := newPrinter()
	p.wrapErrs = true
	p.doPrintf(format, a)
	s := string(p.buf)
	var err error
	switch len(p.wrappedErrs) {
	case 0:
		e := errors.New(s)
		// Remove the extra frame from fmt.Errorf
		if st, ok := e.(interface{ RemoveStackFrames(int) }); ok {
			st.RemoveStackFrames(1)
		}
		err = e
	case 1:
		w := &wrapError{msg: s}
		w.err, _ = a[p.wrappedErrs[0]].(error)
		// Capture stack if inner error needs stack
		if errors.ShouldCaptureStack(w) {
			w.Stack = errors.CaptureStack(1)
		}
		err = w
	default:
		if p.reordered {
			slices.Sort(p.wrappedErrs)
		}
		var errs []error
		for i, argNum := range p.wrappedErrs {
			if i > 0 && p.wrappedErrs[i-1] == argNum {
				continue
			}
			if e, ok := a[argNum].(error); ok {
				errs = append(errs, e)
			}
		}
		we := &wrapErrors{msg: s, errs: errs}
		// Capture stack if any branch is missing stack
		if errors.ShouldCaptureStack(we) {
			we.Stack = errors.CaptureStack(1)
		}
		err = we
	}
	p.free()
	return err
}

type wrapError struct {
	msg string
	err error
	*errors.Stack
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

type wrapErrors struct {
	msg  string
	errs []error
	*errors.Stack
}

func (e *wrapErrors) Error() string {
	return e.msg
}

func (e *wrapErrors) Unwrap() []error {
	return e.errs
}
