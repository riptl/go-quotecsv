package csv

// A Column of a CSV row.
type Column struct {
	// When used with a Reader, signals whether the field was quoted.
	// When used with a Writer,
	Quoted bool
	// Value of the column
	Value string
}

func (c *Column) String() string {
	return c.Value
}
