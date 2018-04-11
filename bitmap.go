package memsize

import "math/bits"

const (
	uintptrBits  = 32 << (uint64(^uintptr(0)) >> 63)
	uintptrBytes = uintptrBits / 8
	bmBlockRange = 1 * 1024 * 1024 // bytes covered by bmBlock
	bmBlockWords = bmBlockRange / uintptrBits
)

// bitmap is a sparse bitmap.
type bitmap struct {
	blocks map[uintptr]*bmBlock
}

func newBitmap() *bitmap {
	return &bitmap{make(map[uintptr]*bmBlock)}
}

// markRange sets n consecutive bits starting at addr.
func (b *bitmap) markRange(addr, n uintptr) {
	for end := addr + n; addr < end; {
		block, start := b.block(addr)
		for i := addr - start; i < bmBlockRange && addr < end; i++ {
			block.mark(i)
			addr++
		}
	}
}

// isMarked returns the value of the bit at the given address.
func (b *bitmap) isMarked(addr uintptr) bool {
	block, start := b.block(addr)
	return block.isMarked(addr - start)
}

// block finds the block corresponding to the given memory address.
// It also returns the block's starting address.
func (b *bitmap) block(addr uintptr) (*bmBlock, uintptr) {
	index := addr / bmBlockRange
	start := index * bmBlockRange
	block := b.blocks[index]
	if block == nil {
		block = new(bmBlock)
		b.blocks[index] = block
	}
	return block, start
}

// size returns the sum of the byte sizes of all blocks.
func (b *bitmap) size() uintptr {
	return uintptr(len(b.blocks)) * bmBlockWords * uintptrBytes
}

// utilization returns the mean percentage of one bits across all blocks.
func (b *bitmap) utilization() float32 {
	var avg float32
	for _, block := range b.blocks {
		avg += float32(block.onesCount()) / float32(bmBlockRange)
	}
	return avg / float32(len(b.blocks))
}

// bmBlock is a bitmap block.
type bmBlock [bmBlockWords]uintptr

// mark sets the i'th bit to one.
func (b *bmBlock) mark(i uintptr) {
	b[i/uintptrBits] |= 1 << (i % uintptrBits)
}

// isMarked returns the value of the i'th bit.
func (b *bmBlock) isMarked(i uintptr) bool {
	return (b[i/uintptrBits] & (1 << (i % uintptrBits))) != 0
}

// onesCount returns the number of one bits in the block
func (b *bmBlock) onesCount() int {
	var count int
	for _, w := range b {
		count += bits.OnesCount64(uint64(w))
	}
	return count
}
