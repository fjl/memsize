package memsize

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestBitmapBlock(t *testing.T) {
	marks := map[uintptr]bool{
		10:  true,
		13:  true,
		44:  true,
		128: true,
		129: true,
		256: true,
		700: true,
	}
	var b bmBlock
	for i := range marks {
		b.mark(i)
	}
	for i := uintptr(0); i < bmBlockRange; i++ {
		if b.isMarked(i) && !marks[i] {
			t.Fatalf("wrong mark at %d", i)
		}
	}
	if count := b.onesCount(); count != len(marks) {
		t.Fatalf("wrong onesCount: got %d, want %d", count, len(marks))
	}
}

func TestBitmapMarkRange(t *testing.T) {
	N := 1000

	// Generate random mark ranges
	r := rand.New(rand.NewSource(312321312))
	bm := newBitmap()
	ranges := make(map[uintptr]uintptr)
	addr := uintptr(0)
	for i := 0; i < N; i++ {
		addr += uintptr(r.Intn(bmBlockRange))
		len := uintptr(r.Intn(40))
		ranges[addr] = len
		bm.markRange(addr, len)
	}

	// Check all marks are set.
	for start, len := range ranges {
		for i := uintptr(0); i < len; i++ {
			if !bm.isMarked(start + i) {
				t.Fatalf("not marked at %d", start)
			}
		}
	}

	// Probe random addresses.
	for i := 0; i < N; i++ {
		addr := uintptr(r.Uint64())
		marked := false
		for start, len := range ranges {
			if addr >= start && addr < start+len {
				marked = true
				break
			}
		}
		if bm.isMarked(addr) && !marked {
			t.Fatalf("extra mark at %d", addr)
		}
	}
}

func BenchmarkBitmapMarkRange(b *testing.B) {
	var addrs [2048]uintptr
	r := rand.New(rand.NewSource(423098209802))
	for i := range addrs {
		addrs[i] = uintptr(r.Uint64())
	}

	doit := func(b *testing.B, rlen int) {
		bm := newBitmap()
		for i := 0; i < b.N; i++ {
			addr := addrs[i%len(addrs)]
			bm.markRange(addr, uintptr(rlen))
		}
	}
	for rlen := 1; rlen <= 4096; rlen *= 8 {
		b.Run(fmt.Sprintf("%d", rlen), func(b *testing.B) { doit(b, rlen) })
	}
}
