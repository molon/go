// errorcheck

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Check that we don't print duplicate errors for string ->
// array-literal conversion

package main

func main() {
	_ = []byte{"foo"}   // ERROR "cannot use"
	_ = []int{"foo"}    // ERROR "cannot use"
	_ = []rune{"foo"}   // ERROR "cannot use"
	_ = []string{"foo"} // OK
}
