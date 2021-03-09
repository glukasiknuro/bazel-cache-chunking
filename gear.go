package main

import "math/rand"

const (
	// Split mask has 20 rightmost bits set to 1, which correspond to 1M possible values,
	// or consequently, to 1MiB average block size.
	splitMask = ^uint32(0) &^ ((1 << (32 - 20)) - 1)

	// Initial hash. Pseudo random - first digits of square root of 2020.
	initialHash = 0x2cf1c4dc

	// Minimum and maximum chunk sizes, to avoid degenerative chunks (very small or huge).
	minChunkSize = 1 << 16 // 64 KiB
	maxChunkSize = 1 << 24 // 16 MiB
)

var (
	byteVal [256]uint32
)

func init() {
	r := rand.New(rand.NewSource(42))
	for i := 0; i < len(byteVal); i++ {
		byteVal[i] = r.Uint32()
	}
}

// Gear hashing algorithm, see
// https://en.wikipedia.org/wiki/Rolling_hash#Gear_fingerprint_and_content-based_chunking_algorithm_FastCDC
type GearRollingHash struct {
	h       uint32
	written int
}

func NewGearHash() GearRollingHash {
	return GearRollingHash{initialHash, 0}
}

func (h *GearRollingHash) FindSplit(data []byte) int {
	for i := 0; i < len(data); i++ {
		h.h = (h.h << 1) + byteVal[data[i]]
		h.written++
		if h.written >= maxChunkSize {
			return i + 1
		}
		if h.written >= minChunkSize && (h.h&splitMask == 0) {
			return i + 1
		}
	}
	return -1
}
