// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csv

import (
	"io"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRead(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output [][]Column
		Error  error

		// These fields are copied into the Reader
		Comma              rune
		Comment            rune
		UseFieldsPerRecord bool // false (default) means FieldsPerRecord is -1
		FieldsPerRecord    int
		LazyQuotes         bool
		TrimLeadingSpace   bool
		ReuseRecord        bool
	}{{
		Name:   "Simple",
		Input:  "a,b,c\n",
		Output: [][]Column{{c("a"), c("b"), c("c")}},
	}, {
		Name:   "CRLF",
		Input:  "a,b\r\nc,d\r\n",
		Output: [][]Column{{c("a"), c("b")}, {c("c"), c("d")}},
	}, {
		Name:   "BareCR",
		Input:  "a,b\rc,d\r\n",
		Output: [][]Column{{c("a"), c("b\rc"), c("d")}},
	}, {
		Name: "RFC4180test",
		Input: `#field1,field2,field3
"aaa","bb
b","ccc"
"a,a","b""bb","ccc"
zzz,yyy,xxx
`,
		Output: [][]Column{
			{c("#field1"), c("field2"), c("field3")},
			{q("aaa"), q("bb\nb"), q("ccc")},
			{q("a,a"), q(`b"bb`), q("ccc")},
			{c("zzz"), c("yyy"), c("xxx")},
		},
		UseFieldsPerRecord: true,
		FieldsPerRecord:    0,
	}, {
		Name:   "NoEOLTest",
		Input:  "a,b,c",
		Output: [][]Column{{c("a"), c("b"), c("c")}},
	}, {
		Name:   "Semicolon",
		Input:  "a;b;c\n",
		Output: [][]Column{{c("a"), c("b"), c("c")}},
		Comma:  ';',
	}, {
		Name: "MultiLine",
		Input: `"two
line","one line","three
line
field"`,
		Output: [][]Column{{q("two\nline"), q("one line"), q("three\nline\nfield")}},
	}, {
		Name:  "BlankLine",
		Input: "a,b,c\n\nd,e,f\n\n",
		Output: [][]Column{
			{c("a"), c("b"), c("c")},
			{c("d"), c("e"), c("f")},
		},
	}, {
		Name:  "BlankLineFieldCount",
		Input: "a,b,c\n\nd,e,f\n\n",
		Output: [][]Column{
			{c("a"), c("b"), c("c")},
			{c("d"), c("e"), c("f")},
		},
		UseFieldsPerRecord: true,
		FieldsPerRecord:    0,
	}, {
		Name:             "TrimSpace",
		Input:            " a,  b,   c\n",
		Output:           [][]Column{{c("a"), c("b"), c("c")}},
		TrimLeadingSpace: true,
	}, {
		Name:   "LeadingSpace",
		Input:  " a,  b,   c\n",
		Output: [][]Column{{c(" a"), c("  b"), c("   c")}},
	}, {
		Name:    "Comment",
		Input:   "#1,2,3\na,b,c\n#comment",
		Output:  [][]Column{{c("a"), c("b"), c("c")}},
		Comment: '#',
	}, {
		Name:   "NoComment",
		Input:  "#1,2,3\na,b,c",
		Output: [][]Column{{c("#1"), c("2"), c("3")}, {c("a"), c("b"), c("c")}},
	}, {
		Name:       "LazyQuotes",
		Input:      `a "word","1"2",a","b`,
		Output:     [][]Column{{c(`a "word"`), q(`1"2`), c(`a"`), c(`b`)}},
		LazyQuotes: true,
	}, {
		Name:       "BareQuotes",
		Input:      `a "word","1"2",a"`,
		Output:     [][]Column{{c(`a "word"`), q(`1"2`), c(`a"`)}},
		LazyQuotes: true,
	}, {
		Name:       "BareDoubleQuotes",
		Input:      `a""b,c`,
		Output:     [][]Column{{c(`a""b`), c(`c`)}},
		LazyQuotes: true,
	}, {
		Name:  "BadDoubleQuotes",
		Input: `a""b,c`,
		Error: &ParseError{StartLine: 1, Line: 1, Column: 1, Err: ErrBareQuote},
	}, {
		Name:             "TrimQuote",
		Input:            ` "a"," b",c`,
		Output:           [][]Column{{q("a"), q(" b"), c("c")}},
		TrimLeadingSpace: true,
	}, {
		Name:  "BadBareQuote",
		Input: `a "word","b"`,
		Error: &ParseError{StartLine: 1, Line: 1, Column: 2, Err: ErrBareQuote},
	}, {
		Name:  "BadTrailingQuote",
		Input: `"a word",b"`,
		Error: &ParseError{StartLine: 1, Line: 1, Column: 10, Err: ErrBareQuote},
	}, {
		Name:  "ExtraneousQuote",
		Input: `"a "word","b"`,
		Error: &ParseError{StartLine: 1, Line: 1, Column: 3, Err: ErrQuote},
	}, {
		Name:               "BadFieldCount",
		Input:              "a,b,c\nd,e",
		Error:              &ParseError{StartLine: 2, Line: 2, Err: ErrFieldCount},
		UseFieldsPerRecord: true,
		FieldsPerRecord:    0,
	}, {
		Name:               "BadFieldCount1",
		Input:              `a,b,c`,
		Error:              &ParseError{StartLine: 1, Line: 1, Err: ErrFieldCount},
		UseFieldsPerRecord: true,
		FieldsPerRecord:    2,
	}, {
		Name:   "FieldCount",
		Input:  "a,b,c\nd,e",
		Output: [][]Column{{c("a"), c("b"), c("c")}, {c("d"), c("e")}},
	}, {
		Name:   "TrailingCommaEOF",
		Input:  "a,b,c,",
		Output: [][]Column{{c("a"), c("b"), c("c"), c("")}},
	}, {
		Name:   "TrailingCommaEOL",
		Input:  "a,b,c,\n",
		Output: [][]Column{{c("a"), c("b"), c("c"), c("")}},
	}, {
		Name:             "TrailingCommaSpaceEOF",
		Input:            "a,b,c, ",
		Output:           [][]Column{{c("a"), c("b"), c("c"), c("")}},
		TrimLeadingSpace: true,
	}, {
		Name:             "TrailingCommaSpaceEOL",
		Input:            "a,b,c, \n",
		Output:           [][]Column{{c("a"), c("b"), c("c"), c("")}},
		TrimLeadingSpace: true,
	}, {
		Name:             "TrailingCommaLine3",
		Input:            "a,b,c\nd,e,f\ng,hi,",
		Output:           [][]Column{{c("a"), c("b"), c("c")}, {c("d"), c("e"), c("f")}, {c("g"), c("hi"), c("")}},
		TrimLeadingSpace: true,
	}, {
		Name:   "NotTrailingComma3",
		Input:  "a,b,c, \n",
		Output: [][]Column{{c("a"), c("b"), c("c"), c(" ")}},
	}, {
		Name: "CommaFieldTest",
		Input: `x,y,z,w
x,y,z,
x,y,,
x,,,
,,,
"x","y","z","w"
"x","y","z",""
"x","y","",""
"x","","",""
"","","",""
`,
		Output: [][]Column{
			{c("x"), c("y"), c("z"), c("w")},
			{c("x"), c("y"), c("z"), c("")},
			{c("x"), c("y"), c(""), c("")},
			{c("x"), c(""), c(""), c("")},
			{c(""), c(""), c(""), c("")},
			{q("x"), q("y"), q("z"), q("w")},
			{q("x"), q("y"), q("z"), q("")},
			{q("x"), q("y"), q(""), q("")},
			{q("x"), q(""), q(""), q("")},
			{q(""), q(""), q(""), q("")},
		},
	}, {
		Name:  "TrailingCommaIneffective1",
		Input: "a,b,\nc,d,e",
		Output: [][]Column{
			{c("a"), c("b"), c("")},
			{c("c"), c("d"), c("e")},
		},
		TrimLeadingSpace: true,
	}, {
		Name:  "ReadAllReuseRecord",
		Input: "a,b\nc,d",
		Output: [][]Column{
			{c("a"), c("b")},
			{c("c"), c("d")},
		},
		ReuseRecord: true,
	}, {
		Name:  "StartLine1", // Issue 19019
		Input: "a,\"b\nc\"d,e",
		Error: &ParseError{StartLine: 1, Line: 2, Column: 1, Err: ErrQuote},
	}, {
		Name:  "StartLine2",
		Input: "a,b\n\"d\n\n,e",
		Error: &ParseError{StartLine: 2, Line: 5, Column: 0, Err: ErrQuote},
	}, {
		Name:  "CRLFInQuotedField", // Issue 21201
		Input: "A,\"Hello\r\nHi\",B\r\n",
		Output: [][]Column{
			{c("A"), q("Hello\nHi"), c("B")},
		},
	}, {
		Name:   "BinaryBlobField", // Issue 19410
		Input:  "x09\x41\xb4\x1c,aktau",
		Output: [][]Column{{c("x09A\xb4\x1c"), c("aktau")}},
	}, {
		Name:   "TrailingCR",
		Input:  "field1,field2\r",
		Output: [][]Column{{c("field1"), c("field2")}},
	}, {
		Name:   "QuotedTrailingCR",
		Input:  "\"field\"\r",
		Output: [][]Column{{q("field")}},
	}, {
		Name:  "QuotedTrailingCRCR",
		Input: "\"field\"\r\r",
		Error: &ParseError{StartLine: 1, Line: 1, Column: 6, Err: ErrQuote},
	}, {
		Name:   "FieldCR",
		Input:  "field\rfield\r",
		Output: [][]Column{{c("field\rfield")}},
	}, {
		Name:   "FieldCRCR",
		Input:  "field\r\rfield\r\r",
		Output: [][]Column{{c("field\r\rfield\r")}},
	}, {
		Name:   "FieldCRCRLF",
		Input:  "field\r\r\nfield\r\r\n",
		Output: [][]Column{{c("field\r")}, {c("field\r")}},
	}, {
		Name:   "FieldCRCRLFCR",
		Input:  "field\r\r\n\rfield\r\r\n\r",
		Output: [][]Column{{c("field\r")}, {c("\rfield\r")}},
	}, {
		Name:   "FieldCRCRLFCRCR",
		Input:  "field\r\r\n\r\rfield\r\r\n\r\r",
		Output: [][]Column{{c("field\r")}, {c("\r\rfield\r")}, {c("\r")}},
	}, {
		Name:  "MultiFieldCRCRLFCRCR",
		Input: "field1,field2\r\r\n\r\rfield1,field2\r\r\n\r\r,",
		Output: [][]Column{
			{c("field1"), c("field2\r")},
			{c("\r\rfield1"), c("field2\r")},
			{c("\r\r"), c("")},
		},
	}, {
		Name:             "NonASCIICommaAndComment",
		Input:            "a£b,c£ \td,e\n€ comment\n",
		Output:           [][]Column{{c("a"), c("b,c"), c("d,e")}},
		TrimLeadingSpace: true,
		Comma:            '£',
		Comment:          '€',
	}, {
		Name:    "NonASCIICommaAndCommentWithQuotes",
		Input:   "a€\"  b,\"€ c\nλ comment\n",
		Output:  [][]Column{{c("a"), q("  b,"), c(" c")}},
		Comma:   '€',
		Comment: 'λ',
	}, {
		// λ and θ start with the same byte.
		// This tests that the parser doesn't confuse such characters.
		Name:    "NonASCIICommaConfusion",
		Input:   "\"abθcd\"λefθgh",
		Output:  [][]Column{{q("abθcd"), c("efθgh")}},
		Comma:   'λ',
		Comment: '€',
	}, {
		Name:    "NonASCIICommentConfusion",
		Input:   "λ\nλ\nθ\nλ\n",
		Output:  [][]Column{{c("λ")}, {c("λ")}, {c("λ")}},
		Comment: 'θ',
	}, {
		Name:   "QuotedFieldMultipleLF",
		Input:  "\"\n\n\n\n\"",
		Output: [][]Column{{q("\n\n\n\n")}},
	}, {
		Name:  "MultipleCRLF",
		Input: "\r\n\r\n\r\n\r\n",
	}, {
		// The implementation may read each line in several chunks if it doesn't fit entirely
		// in the read buffer, so we should test the code to handle that condition.
		Name:    "HugeLines",
		Input:   strings.Repeat("#ignore\n", 10000) + strings.Repeat("@", 5000) + "," + strings.Repeat("*", 5000),
		Output:  [][]Column{{c(strings.Repeat("@", 5000)), c(strings.Repeat("*", 5000))}},
		Comment: '#',
	}, {
		Name:  "QuoteWithTrailingCRLF",
		Input: "\"foo\"bar\"\r\n",
		Error: &ParseError{StartLine: 1, Line: 1, Column: 4, Err: ErrQuote},
	}, {
		Name:       "LazyQuoteWithTrailingCRLF",
		Input:      "\"foo\"bar\"\r\n",
		Output:     [][]Column{{q(`foo"bar`)}},
		LazyQuotes: true,
	}, {
		Name:   "DoubleQuoteWithTrailingCRLF",
		Input:  "\"foo\"\"bar\"\r\n",
		Output: [][]Column{{q(`foo"bar`)}},
	}, {
		Name:   "EvenQuotes",
		Input:  `""""""""`,
		Output: [][]Column{{q(`"""`)}},
	}, {
		Name:  "OddQuotes",
		Input: `"""""""`,
		Error: &ParseError{StartLine: 1, Line: 1, Column: 7, Err: ErrQuote},
	}, {
		Name:       "LazyOddQuotes",
		Input:      `"""""""`,
		Output:     [][]Column{{c(`"""`)}},
		LazyQuotes: true,
	}, {
		Name:  "BadComma1",
		Comma: '\n',
		Error: errInvalidDelim,
	}, {
		Name:  "BadComma2",
		Comma: '\r',
		Error: errInvalidDelim,
	}, {
		Name:  "BadComma3",
		Comma: '"',
		Error: errInvalidDelim,
	}, {
		Name:  "BadComma4",
		Comma: utf8.RuneError,
		Error: errInvalidDelim,
	}, {
		Name:    "BadComment1",
		Comment: '\n',
		Error:   errInvalidDelim,
	}, {
		Name:    "BadComment2",
		Comment: '\r',
		Error:   errInvalidDelim,
	}, {
		Name:    "BadComment3",
		Comment: utf8.RuneError,
		Error:   errInvalidDelim,
	}, {
		Name:    "BadCommaComment",
		Comma:   'X',
		Comment: 'X',
		Error:   errInvalidDelim,
	}}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			r := NewReader(strings.NewReader(tt.Input))

			if tt.Comma != 0 {
				r.Comma = tt.Comma
			}
			r.Comment = tt.Comment
			if tt.UseFieldsPerRecord {
				r.FieldsPerRecord = tt.FieldsPerRecord
			} else {
				r.FieldsPerRecord = -1
			}
			r.LazyQuotes = tt.LazyQuotes
			r.TrimLeadingSpace = tt.TrimLeadingSpace
			r.ReuseRecord = tt.ReuseRecord

			out, err := r.ReadAll()
			if !reflect.DeepEqual(err, tt.Error) {
				t.Errorf("ReadAll() error:\ngot  %v\nwant %v", err, tt.Error)
			} else if !reflect.DeepEqual(out, tt.Output) {
				t.Errorf("ReadAll() output:\ngot  %v\nwant %v", out, tt.Output)
			}
		})
	}
}

// nTimes is an io.Reader which yields the string s n times.
type nTimes struct {
	s   string
	n   int
	off int
}

func (r *nTimes) Read(p []byte) (n int, err error) {
	for {
		if r.n <= 0 || r.s == "" {
			return n, io.EOF
		}
		n0 := copy(p, r.s[r.off:])
		p = p[n0:]
		n += n0
		r.off += n0
		if r.off == len(r.s) {
			r.off = 0
			r.n--
		}
		if len(p) == 0 {
			return
		}
	}
}

// benchmarkRead measures reading the provided CSV rows data.
// initReader, if non-nil, modifies the Reader before it's used.
func benchmarkRead(b *testing.B, initReader func(*Reader), rows string) {
	b.ReportAllocs()
	r := NewReader(&nTimes{s: rows, n: b.N})
	if initReader != nil {
		initReader(r)
	}
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

const benchmarkCSVData = `x,y,z,w
x,y,z,
x,y,,
x,,,
,,,
"x","y","z","w"
"x","y","z",""
"x","y","",""
"x","","",""
"","","",""
`

func BenchmarkRead(b *testing.B) {
	benchmarkRead(b, nil, benchmarkCSVData)
}

func BenchmarkReadWithFieldsPerRecord(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.FieldsPerRecord = 4 }, benchmarkCSVData)
}

func BenchmarkReadWithoutFieldsPerRecord(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.FieldsPerRecord = -1 }, benchmarkCSVData)
}

func BenchmarkReadLargeFields(b *testing.B) {
	benchmarkRead(b, nil, strings.Repeat(`xxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvv
,,zzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
`, 3))
}

func BenchmarkReadReuseRecord(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.ReuseRecord = true }, benchmarkCSVData)
}

func BenchmarkReadReuseRecordWithFieldsPerRecord(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.ReuseRecord = true; r.FieldsPerRecord = 4 }, benchmarkCSVData)
}

func BenchmarkReadReuseRecordWithoutFieldsPerRecord(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.ReuseRecord = true; r.FieldsPerRecord = -1 }, benchmarkCSVData)
}

func BenchmarkReadReuseRecordLargeFields(b *testing.B) {
	benchmarkRead(b, func(r *Reader) { r.ReuseRecord = true }, strings.Repeat(`xxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvv
,,zzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
`, 3))
}

func unboxCols(rows [][]Column) [][]string {
	if rows == nil {
		return nil
	}
	s := make([][]string, len(rows))
	for i, row := range rows {
		s[i] = make([]string, len(row))
		for j, col := range row {
			s[i][j] = col.Value
		}
	}
	return s
}

// Box string in bare column
func c(s string) Column {
	return Column{
		Value:  s,
		Quoted: false,
	}
}

// Box string in quoted column
func q(s string) Column {
	return Column{
		Value:  s,
		Quoted: true,
	}
}
