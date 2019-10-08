package csv

// A Column of a CSV row.
type Column struct {
	// When used with a Reader, signals whether the field was quoted.
	// When sent as true to a Writer, forces quotation.
	Quoted bool
	// Value of the column
	Value string
}

func (c *Column) String() string {
	return c.Value
}
