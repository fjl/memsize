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
	i := st.findspan(uintptr(addr))
	return i < len(st.spans) && st.spans[i].contains(uintptr(addr))
}

// insert adds a span. It returns a tree containing all previous spans
// overlapped by the new one.
func (st *memSpans) insert(start, length uintptr) memSpans {
	if length == 0 {
		return memSpans{}
	}

	newspan := memSpan{start: start, end: start + length}
	i := st.findspan(start)
	if i >= len(st.spans) {
		// New span starts beyond any known span.
		st.spans = append(st.spans, newspan)
		return memSpans{}
	}
	// New span starts inside or before a known span.
	// To insert it, merge with all overlapping known spans.
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
	merged := memSpans{spans: make([]memSpan, mend-mstart)}
	copy(merged.spans, st.spans[mstart:mend])
	st.spans = append(st.spans[:mstart], append([]memSpan{newspan}, st.spans[mend:]...)...)
	return merged
}

func (st *memSpans) findspan(addr uintptr) int {
	return sort.Search(len(st.spans), func(i int) bool {
		return addr <= st.spans[i].start || addr <= st.spans[i].end
	})
}
