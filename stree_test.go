package memsize

import (
	"fmt"
	"sort"
	"testing"
	"testing/quick"
)

func TestSliceTree(t *testing.T) {
	var st sliceTree
	st.insert(0x1, 2)
	st.insert(0x3, 5)
	st.insert(0x1, 3)
	st.insert(0x1, 4)
	t.Log("tree:", st)

	if len(st.arrays) != 1 {
		t.Error("want one array, have", len(st.arrays))
	}
	want := []address{0x5}
	for _, addr := range want {
		if !st.contains(addr) {
			t.Errorf("tree doesn't contain addr %#x", addr)
		}
	}
}

func TestSliceTreeOverlap(t *testing.T) {
	var st sliceTree
	st.insert(0x10, 16)
	st.insert(0x30, 16)
	overlap := st.insert(0x5, 256)
	t.Log("tree:", st)
	t.Log("overlap:", overlap)

	want := []address{0x10, 0x11, 0x30, 0x31}
	wantNot := []address{0x20, 0x21, 0x50}
	for _, addr := range want {
		if !overlap.contains(addr) {
			t.Errorf("overlap tree doesn't contain addr %#x", addr)
		}
	}
	for _, addr := range wantNot {
		if overlap.contains(addr) {
			t.Errorf("overlap tree contains addr %#x, but shouldn't", addr)
		}
	}
}

// This test adds random arrays into the tree and checks whether tree elements are sorted
// and non-overlapping after each insert.
func TestSliceTreeSorted(t *testing.T) {
	const maxUintptr = ^uintptr(0)
	type slice struct {
		Start, Len uintptr
	}

	check := func(input []slice) bool {
		var st sliceTree
		for _, s := range input {
			if s.Len > maxUintptr-s.Start {
				// Avoid overflow.
				s.Len = maxUintptr - s.Start
			}
			e := fmt.Sprintf("{start:%#x, len:%#x}", s.Start, s.Len)
			pre := st.String()
			overlap := st.insert(s.Start, s.Len)
			sorted, disjoint := checkTreeSorted(st)
			if !sorted {
				t.Logf("not sorted after inserting %s into %s", e, pre)
				return false
			}
			if !disjoint {
				t.Logf("st.arrays not disjoint after inserting %s into %s", e, pre)
				return false
			}
			overlapSorted, overlapDisjoint := checkTreeSorted(overlap)
			if !overlapSorted {
				t.Logf("overlap tree %v not sorted after inserting %s into %s", overlap, e, pre)
				return false
			}
			if !overlapDisjoint {
				t.Logf("overlap tree %v not disjoint after inserting %s into %s", overlap, e, pre)
				return false
			}
		}
		return true
	}

	if err := quick.Check(check, nil); err != nil {
		t.Fatal(err)
	}
}

func checkTreeSorted(st sliceTree) (bool, bool) {
	sorted := sort.SliceIsSorted(st.arrays, func(i, j int) bool {
		return st.arrays[i].start < st.arrays[j].start
	})
	disjoint := true
	if len(st.arrays) > 0 {
		prev := st.arrays[0]
		for _, a := range st.arrays[1:] {
			if a.overlaps(prev) {
				disjoint = false
				break
			}
			prev = a
		}
	}
	return sorted, disjoint
}
