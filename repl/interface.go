package repl

// Iterators return each item they contain and reset themselves when completed.
// Reset can be called before the end if going back to the beginneing is needed.
type Iterator interface {
	Next() Sexp
	Reset()
}

// IteratorWrapper returns an iterator for itself.
// This is eaiser for things like slices where it is cleaner to not have the counters on the slice itself
type IteratorWrapper interface {
	WrapIterator() Iterator
}
