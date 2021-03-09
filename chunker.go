package main

type ChunkerType int

const (
	NoChunk = iota
	OneMb
	Gear
)

func (c ChunkerType) String() string {
	return [...]string{"No chunking", "1MB", "Gear"}[c]
}

type Chunker interface {
	// Returns number of bytes from b to split on, or -1 if there is no split yet.
	// If there is no split, this method may be called again with another
	// slice of bytes.
	NextSplit(b []byte) int
}

func NewChunker(t ChunkerType) Chunker {
	switch t {
	case NoChunk:
		return &noChunker{}
	case OneMb:
		return &oneMbChunker{}
	case Gear:
		return &gearChunker{h: NewGearHash()}
	default:
		panic(t)
	}
}

type noChunker struct{}

func (c *noChunker) NextSplit(b []byte) int {
	return -1
}

type oneMbChunker struct {
	written int
}

func (c *oneMbChunker) NextSplit(b []byte) int {
	if c.written+len(b) >= (1 << 20) {
		splitBytes := (1 << 20) - c.written
		return splitBytes
	}
	c.written += len(b)
	return -1
}

type gearChunker struct {
	h GearRollingHash
}

func (c *gearChunker) NextSplit(b []byte) int {
	return c.h.FindSplit(b)
}
