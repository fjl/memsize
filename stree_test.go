package memsize

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
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
			t.Errorf("tree doesn't contain addr %v", addr)
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
			t.Errorf("overlap tree doesn't contain addr %v", addr)
		}
	}
	for _, addr := range wantNot {
		if overlap.contains(addr) {
			t.Errorf("overlap tree contains addr %v, but shouldn't", addr)
		}
	}
}

type testSlice struct {
	start, len uintptr
}

const maxUintptr = ^uintptr(0)

func (testSlice) Generate(rand *rand.Rand, size int) reflect.Value {
	var s testSlice
	s.start = uintptr(rand.Intn(size))
	limit := maxUintptr - s.start
	s.len = uintptr(rand.Intn(size)) % limit
	return reflect.ValueOf(s)
}

func (s testSlice) String() string {
	return fmt.Sprintf("{start: %#x, len: %#x}", s.start, s.len)
}

// This test adds random arrays into the tree and checks whether tree elements are sorted
// and non-overlapping after each insert.
func TestSliceTreeSorted(t *testing.T) {
	check := func(input []testSlice) bool {
		var st sliceTree
		for _, s := range input {
			pre := st.String()
			overlap := st.insert(s.start, s.len)
			if err := st.checkConsistency(); err != nil {
				t.Logf("%v after inserting %v into %s", err, s, pre)
				return false
			}
			if err := overlap.checkConsistency(); err != nil {
				t.Logf("overlap tree %v %v after inserting %v into %s", overlap, err, s, pre)
				return false
			}
		}
		return true
	}

	if err := quick.Check(check, nil); err != nil {
		t.Fatal(err)
	}
}

func (st sliceTree) checkConsistency() error {
	sorted := sort.SliceIsSorted(st.arrays, func(i, j int) bool {
		return st.arrays[i].start < st.arrays[j].start
	})
	if !sorted {
		return errors.New("not sorted")
	}
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
	if !disjoint {
		return errors.New("not disjoint")
	}
	return nil
}
