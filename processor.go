package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"sync"
	"sync/atomic"
)

type Processor struct {
	ChunkerType     ChunkerType
	Compressor      Compressor
	chunks          sync.Map
	duplicateChunks int64
}

// Returns a writer that should be feed content of a file.
func (p *Processor) NewWriter() io.WriteCloser {
	w := &chunkedWriter{
		p: p,
	}
	w.resetChunker()
	return w
}

func (p *Processor) GetSize() int64 {
	var uniqueSize int64
	p.chunks.Range(func(hash, size interface{}) bool {
		uniqueSize += size.(int64)
		return true
	})
	return uniqueSize
}

func (p *Processor) GetExtraStats() string {
	var chunks int
	p.chunks.Range(func(hash, size interface{}) bool {
		chunks++
		return true
	})
	return fmt.Sprintf("chunks: %d   duplicate: %d", chunks, p.duplicateChunks)
}

type chunkedWriter struct {
	p *Processor
	c Chunker
	h *hashingWriter
}

func (w *chunkedWriter) Write(p []byte) (n int, err error) {
	orgLen := len(p)
	for {
		splitBytes := w.c.NextSplit(p)
		if splitBytes == 0 {
			panic(w.c)
		}

		// May happen at the very beggining, or right after a split.
		if w.h == nil {
			w.resetWriter()
		}

		if splitBytes == -1 {
			// There is no split in current data, wait for next write.
			_, err = w.h.Write(p)
			if err != nil {
				panic(err)
			}
			return orgLen, nil
		} else {
			// There is a split, create chunk and process the rest of bytes.
			_, err = w.h.Write(p[:splitBytes])
			if err != nil {
				panic(err)
			}

			w.chunk()
			w.resetChunker()
			w.h = nil

			p = p[splitBytes:]

			if len(p) == 0 {
				return orgLen, nil
			}
		}
	}
}

func (w *chunkedWriter) Close() error {
	if w.h != nil {
		w.chunk()
	}
	return nil
}

func (w *chunkedWriter) resetWriter() {
	w.h = NewHashingWriter(w.p.Compressor)
}

func (w *chunkedWriter) resetChunker() {
	w.c = NewChunker(w.p.ChunkerType)
}

func (w *chunkedWriter) chunk() {
	h, size := w.h.GetHashAndSize()

	_, loaded := w.p.chunks.LoadOrStore(h, size)
	if loaded {
		atomic.AddInt64(&(w.p.duplicateChunks), 1)
	}
}

// Writer that computes hash of raw data, and size of written data after compression data.
type hashingWriter struct {
	out          io.WriteCloser
	hash         hash.Hash
	bytesWritten int64
}

func NewHashingWriter(compressor Compressor) *hashingWriter {
	h := hashingWriter{}
	h.out = compressor.NewWriter(&CountingWriter{&h.bytesWritten})
	h.hash = sha512.New512_256()
	return &h
}

func (w *hashingWriter) Write(p []byte) (int, error) {
	_, err := w.hash.Write(p)
	if err != nil {
		panic(err)
	}
	return w.out.Write(p)
}

func (w *hashingWriter) GetHashAndSize() (string, int64) {
	err := w.out.Close()
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(w.hash.Sum(nil)), w.bytesWritten
}
