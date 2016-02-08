package zygo

type Sexp interface {
	SexpString() string
}

// Getter gets a value stored at a given key
type Getter interface {
	Get(key Sexp) Sexp
}

// Setter sets a given value and returns a new version of itself that contains that k/v pair
type Setter interface {
	Set(key, val Sexp) Sexp
}

// Indexer return the given index
type Indexer interface {
	Index(index SexpInt) Sexp
}

// Appenders append values to the end of themselves and returna new version of it
type Appender interface {
	Append(val Sexp) Sexp
}

type Prepender interface {
	Prepend(val Sexp) Sexp
}

type Exister interface {
	Exists(v Sexp) Sexp
}

// List is a singly linked list
type List struct {
	next *List
	val  Sexp
}

// Index returns the given index in liniar time
func (l *List) Index(index SexpInt) Sexp {
	i := 0
	x := int(SexpInt)
	y := l
	for i != x {
		y = l.next
	}

	return SexpNull
}
