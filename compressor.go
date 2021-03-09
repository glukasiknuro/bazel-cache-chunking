package main

import (
	"compress/gzip"
	zstdcgo "github.com/DataDog/zstd"
	"github.com/klauspost/compress/zstd"
	"io"
	"sync"
)

type Compressor int

const (
	Identity = iota
	ZstdDefault
	ZstdSpeed
	ZstdCgoDefault
	ZstdCgoSpeed
	Gzip
)

func (c Compressor) String() string {
	return [...]string{"IDENTITY", "ZSTD DEFAULT", "ZSTD BEST_SPEED", "ZSTD CGO DEFAULT", "ZSTD CGO SPEED", "GZIP"}[c]
}

var zstdDefaultPool = sync.Pool{
	New: func() interface{} {
		w, err := zstd.NewWriter(nil)
		if err != nil {
			panic(err)
		}
		return w
	},
}

var zstdSpeedFastestPool = sync.Pool{
	New: func() interface{} {
		w, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		if err != nil {
			panic(err)
		}
		return w
	},
}

func (c Compressor) NewWriter(w io.WriteCloser) io.WriteCloser {
	switch c {
	case Identity:
		return w
	case ZstdDefault:
		z := zstdDefaultPool.Get().(*zstd.Encoder)
		z.Reset(w)
		return z
	case ZstdSpeed:
		z := zstdSpeedFastestPool.Get().(*zstd.Encoder)
		z.Reset(w)
		return z
	case ZstdCgoSpeed:
		return zstdcgo.NewWriterLevel(w, zstdcgo.BestSpeed)
	case ZstdCgoDefault:
		return zstdcgo.NewWriterLevel(w, zstdcgo.DefaultCompression)
	case Gzip:
		return gzip.NewWriter(w)
	default:
		panic(c)
	}
}
