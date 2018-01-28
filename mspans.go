package memsize

import (
	"fmt"
	"sort"
)

// memSpans stores non-overlapping memory spans.
// It is used to track memory regions visited during scan.
type memSpans struct {
	spans []memSpan
}

type memSpan struct {
	start, end uintptr
}

const (
	// These are used as special markers in spanInsert.start.
	insertNoop  = -2
	insertAtEnd = -1
)

type spanInsert struct {
	st         *memSpans
	start, end int // overlap indexes
	newspan    memSpan
}

func (ba memSpan) contains(addr uintptr) bool {
	return addr >= ba.start && addr < ba.end
}

func (ba memSpan) overlaps(other memSpan) bool {
	return ba.start <= other.end && ba.end >= other.start
}

func (ba memSpan) String() string {
	return fmt.Sprintf("[%#x..%#x]", ba.start, ba.end)
}

func (st memSpans) String() string {
	r := "{"
	for i, a := range st.spans {
		r += a.String()
		if i < len(st.spans)-1 {
			r += " "
		}
	}
	return r + "}"
}

// contains reports whether the given address is contained in any known span.
func (st *memSpans) contains(addr address) bool {
	return contains(st.spans, uintptr(addr))
}

func (ins *spanInsert) contains(addr address) bool {
	return ins.st != nil && contains(ins.st.spans[ins.start:ins.end], uintptr(addr))
}

// It returns a tree containing all previous spans overlapped by the new one.
func (st *memSpans) insert(start, length uintptr) spanInsert {
	if length == 0 {
		return spanInsert{start: insertNoop}
	}

	newspan := memSpan{start: start, end: start + length}
	i := findspan(st.spans, start)
	if i >= len(st.spans) {
		return spanInsert{start: insertAtEnd, newspan: newspan}
	}
	// New span starts inside or before a known span.
	// Collect overlapping spans.
	mstart, mend := i, i
	for ; mend < len(st.spans) && newspan.overlaps(st.spans[mend]); mend++ {
		e := st.spans[mend]
		if e.start < newspan.start {
			newspan.start = e.start
		}
		if e.end > newspan.end {
			newspan.end = e.end
		}
	}
	return spanInsert{st: st, start: mstart, end: mend, newspan: newspan}
}

func (st *memSpans) commit(ins spanInsert) {
	switch ins.start {
	case insertNoop:
	case insertAtEnd:
		st.spans = append(st.spans, ins.newspan)
	default:
		st.spans = append(st.spans[:ins.start], append([]memSpan{ins.newspan}, st.spans[ins.end:]...)...)
	}
}

func contains(spans []memSpan, addr uintptr) bool {
	i := findspan(spans, uintptr(addr))
	return i < len(spans) && spans[i].contains(uintptr(addr))
}

func findspan(spans []memSpan, addr uintptr) int {
	return sort.Search(len(spans), func(i int) bool {
		return addr <= spans[i].start || addr <= spans[i].end
	})
}
