// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csv

import (
	"bytes"
	"errors"
	"testing"
)

var writeTests = []struct {
	Input  [][]Column
	Output string
	Error  error
}{
	{Input: [][]Column{{{Value: "abc"}}}, Output: "abc\n"},
	{Input: [][]Column{{{Value: `"abc"`}}}, Output: `"""abc"""` + "\n"},
	{Input: [][]Column{{{Value: `a"b`}}}, Output: `"a""b"` + "\n"},
	{Input: [][]Column{{{Value: `"a"b"`}}}, Output: `"""a""b"""` + "\n"},
	{Input: [][]Column{{{Value: " abc"}}}, Output: `" abc"` + "\n"},
	{Input: [][]Column{{{Value: "abc,def"}}}, Output: `"abc,def"` + "\n"},
	{Input: [][]Column{{{Value: "abc"}, {Value: "def"}}}, Output: "abc,def\n"},
	{Input: [][]Column{{{Value: "abc"}}, {{Value: "def"}}}, Output: "abc\ndef\n"},
	{Input: [][]Column{{{Value: "abc\ndef"}}}, Output: "\"abc\ndef\"\n"},
	{Input: [][]Column{{{Value: "abc\rdef"}}}, Output: "\"abc\rdef\"\n"},
	{Input: [][]Column{{{Value: ""}}}, Output: "\n"},
	{Input: [][]Column{{{Value: ""}, {Value: ""}}}, Output: ",\n"},
	{Input: [][]Column{{{Value: ""}, {Value: ""}, {Value: ""}}}, Output: ",,\n"},
	{Input: [][]Column{{{Value: ""}, {Value: ""}, {Value: "a"}}}, Output: ",,a\n"},
	{Input: [][]Column{{{Value: ""}, {Value: "a"}, {Value: ""}}}, Output: ",a,\n"},
	{Input: [][]Column{{{Value: ""}, {Value: "a"}, {Value: "a"}}}, Output: ",a,a\n"},
	{Input: [][]Column{{{Value: "a"}, {Value: ""}, {Value: ""}}}, Output: "a,,\n"},
	{Input: [][]Column{{{Value: "a"}, {Value: ""}, {Value: "a"}}}, Output: "a,,a\n"},
	{Input: [][]Column{{{Value: "a"}, {Value: "a"}, {Value: ""}}}, Output: "a,a,\n"},
	{Input: [][]Column{{{Value: "a"}, {Value: "a"}, {Value: "a"}}}, Output: "a,a,a\n"},
	{Input: [][]Column{{{Value: `\.`}}}, Output: "\"\\.\"\n"},
	{Input: [][]Column{{{Value: "x09\x41\xb4\x1c"}, {Value: "aktau"}}}, Output: "x09\x41\xb4\x1c,aktau\n"},
	{Input: [][]Column{{{Value: ",x09\x41\xb4\x1c"}, {Value: "aktau"}}}, Output: "\",x09\x41\xb4\x1c\",aktau\n"},
	{Input: [][]Column{{{Value: "abc", Quoted: true}}}, Output: `"abc"` + "\n"},
}

func TestWrite(t *testing.T) {
	for n, tt := range writeTests {
		b := &bytes.Buffer{}
		f := NewWriter(b)

		err := f.WriteAll(tt.Input)
		if err != tt.Error {
			t.Errorf("Unexpected error:\ngot  %v\nwant %v", err, tt.Error)
		}
		out := b.String()
		if out != tt.Output {
			t.Errorf("#%d: out=%q want %q", n, out, tt.Output)
		}
	}
}

type errorWriter struct{}

func (e errorWriter) Write(b []byte) (int, error) {
	return 0, errors.New("test")
}

func TestError(t *testing.T) {
	b := &bytes.Buffer{}
	f := NewWriter(b)
	f.Write([]Column{{Value: "abc"}})
	f.Flush()
	err := f.Error()

	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	f = NewWriter(errorWriter{})
	f.Write([]Column{{Value: "abc"}})
	f.Flush()
	err = f.Error()

	if err == nil {
		t.Error("Error should not be nil")
	}
}
