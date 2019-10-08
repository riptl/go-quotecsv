# go-quotecsv

Fork of Go's `encoding/csv` to offer greater control over quoting,
by adding a new `Column` type:

```go
type Column struct {
	Quoted bool
	Value string
}
```

This is useful when serializing non-literal primitives like NULLs or numbers.

When reading, `Quoted` reports whether a field was quoted or not.
When writing, `Quoted` forces quotation (but not the lack thereof in the opposite case).
