package menmos

// Range represents an end-inclusive byte range.
type Range struct {
	// Start is the start of the range (starting at 0).
	Start int64

	// End is the end of the range.
	End int64
}
