package memsize

import (
	"fmt"
	"sort"
)

// sliceTree stores slice backing arrays and their extent in memory.
type sliceTree struct {
	arrays []backarray
}

type backarray struct {
	start, end uintptr
}

func (ba backarray) contains(addr uintptr) bool {
	return addr >= ba.start && addr < ba.end
}

func (ba backarray) overlaps(other backarray) bool {
	return ba.start <= other.end && ba.end >= other.start
}

func (ba backarray) String() string {
	return fmt.Sprintf("[%#x..%#x]", ba.start, ba.end)
}

func (st sliceTree) String() string {
	r := "{"
	for i, a := range st.arrays {
		r += a.String()
		if i < len(st.arrays)-1 {
			r += " "
		}
	}
	return r + "}"
}

// contains reports whether the given address is contained in any known backing array.
func (st *sliceTree) contains(addr address) bool {
	i := st.findarray(uintptr(addr))
	return i < len(st.arrays) && st.arrays[i].contains(uintptr(addr))
}

// insert adds a slice. It returns a tree containing all previous arrays
// overlapped by the new one.
func (st *sliceTree) insert(start, length uintptr) sliceTree {
	if length == 0 {
		return sliceTree{}
	}

	newarray := backarray{start: start, end: start + length}
	i := st.findarray(start)
	if i >= len(st.arrays) {
		// New array starts beyond any known array.
		st.arrays = append(st.arrays, newarray)
		return sliceTree{}
	}
	// New array starts inside or before a known array.
	// To insert it, merge with all overlapping known arrays.
	mstart, mend := i, i
	for ; mend < len(st.arrays) && newarray.overlaps(st.arrays[mend]); mend++ {
		e := st.arrays[mend]
		if e.start < newarray.start {
			newarray.start = e.start
		}
		if e.end > newarray.end {
			newarray.end = e.end
		}
	}
	merged := sliceTree{arrays: make([]backarray, mend-mstart)}
	copy(merged.arrays, st.arrays[mstart:mend])
	st.arrays = append(st.arrays[:mstart], append([]backarray{newarray}, st.arrays[mend:]...)...)
	return merged
}

func (st *sliceTree) findarray(addr uintptr) int {
	return sort.Search(len(st.arrays), func(i int) bool {
		return addr <= st.arrays[i].start || addr <= st.arrays[i].end
	})
}
