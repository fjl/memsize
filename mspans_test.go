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

func TestMemSpans(t *testing.T) {
	var sp memSpans
	sp.insert(0x1, 2)
	sp.insert(0x3, 5)
	sp.insert(0x1, 3)
	sp.insert(0x1, 4)
	t.Log("tree:", sp)

	if len(sp.spans) != 1 {
		t.Error("want one span, have", len(sp.spans))
	}
	want := []address{0x5}
	for _, addr := range want {
		if !sp.contains(addr) {
			t.Errorf("tree doesn't contain addr %v", addr)
		}
	}
}

func TestMemSpansTreeOverlap(t *testing.T) {
	var sp memSpans
	sp.insert(0x10, 16)
	sp.insert(0x30, 16)
	overlap := sp.insert(0x5, 256)
	t.Log("tree:", sp)
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

type testSpan struct {
	start, len uintptr
}

const maxUintptr = ^uintptr(0)

func (testSpan) Generate(rand *rand.Rand, size int) reflect.Value {
	var s testSpan
	s.start = uintptr(rand.Intn(size))
	limit := maxUintptr - s.start
	s.len = uintptr(rand.Intn(size)) % limit
	return reflect.ValueOf(s)
}

func (s testSpan) String() string {
	return fmt.Sprintf("{start: %#x, len: %#x}", s.start, s.len)
}

// This test adds random spans and checks whether the elements are sorted
// and non-overlapping after each insert.
func TestMemSpansSorted(t *testing.T) {
	check := func(input []testSpan) bool {
		var sp memSpans
		for _, s := range input {
			pre := sp.String()
			overlap := sp.insert(s.start, s.len)
			if err := sp.checkConsistency(); err != nil {
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

func (sp memSpans) checkConsistency() error {
	sorted := sort.SliceIsSorted(sp.spans, func(i, j int) bool {
		return sp.spans[i].start < sp.spans[j].start
	})
	if !sorted {
		return errors.New("not sorted")
	}
	disjoint := true
	if len(sp.spans) > 0 {
		prev := sp.spans[0]
		for _, a := range sp.spans[1:] {
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
