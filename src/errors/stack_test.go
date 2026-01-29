// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// customError is an error without stack trace for testing
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// TestErrorTypeName verifies that error types have the expected names
func TestErrorTypeName(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "errors.New returns *errors.errorString",
			err:      errors.New("test"),
			wantType: "*errors.errorString",
		},
		{
			name:     "errors.Join returns *errors.joinError",
			err:      errors.Join(errors.New("a"), errors.New("b")),
			wantType: "*errors.joinError",
		},
		{
			name:     "fmt.Errorf without %w returns *errors.errorString",
			err:      fmt.Errorf("test %d", 42),
			wantType: "*errors.errorString",
		},
		{
			name:     "fmt.Errorf with %w returns *fmt.wrapError",
			err:      fmt.Errorf("wrap: %w", errors.New("inner")),
			wantType: "*fmt.wrapError",
		},
		{
			name:     "fmt.Errorf with multiple %w returns *fmt.wrapErrors",
			err:      fmt.Errorf("wrap: %w and %w", errors.New("a"), errors.New("b")),
			wantType: "*fmt.wrapErrors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := reflect.TypeOf(tt.err).String()
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
			t.Logf("Type: %s", gotType)
		})
	}
}

// StackTracer interface for testing
type stackTracer interface {
	HasStack() bool
}

// TestErrorsNewHasStack verifies errors.New captures stack
func TestErrorsNewHasStack(t *testing.T) {
	err := errors.New("test error")

	st, ok := err.(stackTracer)
	if !ok {
		t.Fatal("errors.New should return an error implementing HasStack()")
	}
	if !st.HasStack() {
		t.Error("errors.New should capture stack trace")
	}

	// %v should not include stack
	vOutput := fmt.Sprintf("%v", err)
	if vOutput != "test error" {
		t.Errorf("%%v = %q, want %q", vOutput, "test error")
	}

	// %+s should include stack
	plusS := fmt.Sprintf("%+s", err)
	t.Logf("errors.New %%+s output:\n%s", plusS)
	if !strings.Contains(plusS, "test error") {
		t.Error("plusS should contain error message")
	}
	if !strings.Contains(plusS, "TestErrorsNewHasStack") {
		t.Error("plusS should contain function name")
	}
	if !strings.Contains(plusS, ".go:") {
		t.Error("plusS should contain file:line")
	}
}

// TestFmtErrorfNoWrapHasStack verifies fmt.Errorf without %w captures stack
func TestFmtErrorfNoWrapHasStack(t *testing.T) {
	err := fmt.Errorf("formatted: %d", 42)

	st, ok := err.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf without %w should return an error implementing HasStack()")
	}
	if !st.HasStack() {
		t.Error("fmt.Errorf without %w should capture stack trace")
	}

	plusS := fmt.Sprintf("%+s", err)
	t.Logf("fmt.Errorf no %%w, %%+s output:\n%s", plusS)
	if !strings.Contains(plusS, "formatted: 42") {
		t.Error("plusS should contain error message")
	}
}

// TestFmtErrorfWrapStackedInner verifies fmt.Errorf with %w and stacked inner does NOT add stack
func TestFmtErrorfWrapStackedInner(t *testing.T) {
	inner := errors.New("inner")
	wrapped := fmt.Errorf("wrapped: %w", inner)

	st, ok := wrapped.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf with %w should return an error implementing HasStack()")
	}
	// Should NOT have its own stack since inner already has one
	if st.HasStack() {
		t.Error("fmt.Errorf with stacked inner should NOT capture additional stack")
	}

	plusS := fmt.Sprintf("%+s", wrapped)
	t.Logf("fmt.Errorf %%w stacked inner, %%+s output:\n%s", plusS)
}

// TestFmtErrorfWrapUnstackedInner verifies fmt.Errorf with %w and unstacked inner adds stack
func TestFmtErrorfWrapUnstackedInner(t *testing.T) {
	inner := &customError{msg: "no stack"}
	wrapped := fmt.Errorf("wrapped: %w", inner)

	st, ok := wrapped.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf with %w should return an error implementing HasStack()")
	}
	// Should have stack since inner doesn't have one
	if !st.HasStack() {
		t.Error("fmt.Errorf with unstacked inner should capture stack")
	}

	plusS := fmt.Sprintf("%+s", wrapped)
	t.Logf("fmt.Errorf %%w unstacked inner, %%+s output:\n%s", plusS)
	if !strings.Contains(plusS, "TestFmtErrorfWrapUnstackedInner") {
		t.Error("plusS should contain function name")
	}
}

// TestFmtErrorfWrapNilInner verifies fmt.Errorf with %w nil captures stack
func TestFmtErrorfWrapNilInner(t *testing.T) {
	var nilErr error = nil
	wrapped := fmt.Errorf("wrapped: %w", nilErr)

	st, ok := wrapped.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf with %w nil should return an error implementing HasStack()")
	}
	// Should have stack since inner is nil
	if !st.HasStack() {
		t.Error("fmt.Errorf with nil inner should capture stack")
	}

	plusS := fmt.Sprintf("%+s", wrapped)
	t.Logf("fmt.Errorf %%w nil, %%+s output:\n%s", plusS)
}

// TestErrorsJoinAllStacked verifies errors.Join with all stacked errors does NOT add stack
func TestErrorsJoinAllStacked(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	joined := errors.Join(err1, err2)

	st, ok := joined.(stackTracer)
	if !ok {
		t.Fatal("errors.Join should return an error implementing HasStack()")
	}
	// Should NOT have its own stack since all children have stacks
	if st.HasStack() {
		t.Error("errors.Join with all stacked children should NOT capture additional stack")
	}

	plusS := fmt.Sprintf("%+s", joined)
	t.Logf("errors.Join all stacked, %%+s output:\n%s", plusS)
}

// TestErrorsJoinOneUnstacked verifies errors.Join with one unstacked error adds stack
func TestErrorsJoinOneUnstacked(t *testing.T) {
	err1 := errors.New("error 1")
	unstacked := &customError{msg: "no stack"}
	joined := errors.Join(err1, unstacked)

	st, ok := joined.(stackTracer)
	if !ok {
		t.Fatal("errors.Join should return an error implementing HasStack()")
	}
	// Should have stack since one child doesn't have one
	if !st.HasStack() {
		t.Error("errors.Join with unstacked child should capture stack")
	}

	plusS := fmt.Sprintf("%+s", joined)
	t.Logf("errors.Join one unstacked, %%+s output:\n%s", plusS)
	if !strings.Contains(plusS, "TestErrorsJoinOneUnstacked") {
		t.Error("plusS should contain function name")
	}
}

// TestFmtErrorfMultiWrapAllStacked verifies multi-wrap with all stacked does NOT add stack
func TestFmtErrorfMultiWrapAllStacked(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	wrapped := fmt.Errorf("multi: %w and %w", err1, err2)

	st, ok := wrapped.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf with multiple %w should return an error implementing HasStack()")
	}
	// Should NOT have its own stack since all children have stacks
	if st.HasStack() {
		t.Error("fmt.Errorf multi-wrap with all stacked should NOT capture additional stack")
	}

	plusS := fmt.Sprintf("%+s", wrapped)
	t.Logf("fmt.Errorf multi-wrap all stacked, %%+s output:\n%s", plusS)
}

// TestFmtErrorfMultiWrapOneUnstacked verifies multi-wrap with one unstacked adds stack
func TestFmtErrorfMultiWrapOneUnstacked(t *testing.T) {
	err1 := errors.New("error 1")
	unstacked := &customError{msg: "no stack"}
	wrapped := fmt.Errorf("multi: %w and %w", err1, unstacked)

	st, ok := wrapped.(stackTracer)
	if !ok {
		t.Fatal("fmt.Errorf with multiple %w should return an error implementing HasStack()")
	}
	// Should have stack since one child doesn't have one
	if !st.HasStack() {
		t.Error("fmt.Errorf multi-wrap with unstacked should capture stack")
	}

	plusS := fmt.Sprintf("%+s", wrapped)
	t.Logf("fmt.Errorf multi-wrap one unstacked, %%+s output:\n%s", plusS)
}

// TestVFormatDoesNotIncludeStack verifies %v and %s do not include stack
func TestVFormatDoesNotIncludeStack(t *testing.T) {
	err := errors.New("test error")

	vOutput := fmt.Sprintf("%v", err)
	sOutput := fmt.Sprintf("%s", err)

	if strings.Contains(vOutput, ".go:") {
		t.Errorf("%%v should not contain stack trace, got: %s", vOutput)
	}
	if strings.Contains(sOutput, ".go:") {
		t.Errorf("%%s should not contain stack trace, got: %s", sOutput)
	}
	t.Logf("%%v: %s", vOutput)
	t.Logf("%%s: %s", sOutput)
}

// TestStackErrorMethod verifies Error() returns just the message
func TestStackErrorMethod(t *testing.T) {
	err := errors.New("simple message")
	if err.Error() != "simple message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "simple message")
	}
}

// TestUnwrapChain verifies errors.Unwrap works correctly
func TestUnwrapChain(t *testing.T) {
	inner := errors.New("inner")
	wrapped := fmt.Errorf("outer: %w", inner)

	unwrapped := errors.Unwrap(wrapped)
	if unwrapped == nil {
		t.Fatal("Unwrap returned nil")
	}
	if unwrapped.Error() != "inner" {
		t.Errorf("Unwrap().Error() = %q, want %q", unwrapped.Error(), "inner")
	}
}

// TestErrorsIs verifies errors.Is works with stacked errors
func TestErrorsIs(t *testing.T) {
	inner := errors.New("inner")
	wrapped := fmt.Errorf("outer: %w", inner)

	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should find inner error in wrapped error")
	}
}

// TestErrorsAs verifies errors.As works with stacked errors
func TestErrorsAs(t *testing.T) {
	custom := &customError{msg: "custom"}
	wrapped := fmt.Errorf("outer: %w", custom)

	var target *customError
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should find customError in wrapped error")
	}
	if target.msg != "custom" {
		t.Errorf("target.msg = %q, want %q", target.msg, "custom")
	}
}

// TestNestedWrapping verifies deeply nested wrapping works
func TestNestedWrapping(t *testing.T) {
	err1 := errors.New("base")
	err2 := fmt.Errorf("level1: %w", err1)
	err3 := fmt.Errorf("level2: %w", err2)
	err4 := fmt.Errorf("level3: %w", err3)

	plusS := fmt.Sprintf("%+s", err4)
	t.Logf("Nested wrapping %%+s output:\n%s", plusS)

	if !strings.Contains(plusS, "base") {
		t.Error("plusS should contain base error message")
	}
	if !strings.Contains(plusS, "level1") {
		t.Error("plusS should contain level1 message")
	}
	if !strings.Contains(plusS, "level2") {
		t.Error("plusS should contain level2 message")
	}
	if !strings.Contains(plusS, "level3") {
		t.Error("plusS should contain level3 message")
	}
}

// TestErrUnsupportedType verifies errors.ErrUnsupported has correct type
func TestErrUnsupportedType(t *testing.T) {
	gotType := reflect.TypeOf(errors.ErrUnsupported).String()
	if gotType != "*errors.errorString" {
		t.Errorf("errors.ErrUnsupported type = %q, want %q", gotType, "*errors.errorString")
	}
	t.Logf("errors.ErrUnsupported type: %s", gotType)
}
