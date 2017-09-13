package memsize

import (
	"testing"
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
