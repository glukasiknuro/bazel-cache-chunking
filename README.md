# Bazel cache chunking stats

Contains sample binary that scans files in provided directory and display total
size of the files if those were chunked, only unique chunks left, and the chunks compressed.

Written to check gains if data in bazel remote cache was chunked, see e.g.
https://github.com/bazelbuild/remote-apis/issues/178

Available chunking strageties include chunk on 1MiB boundary, and content based
chunking based on
[FastCDC](https://en.wikipedia.org/wiki/Rolling_hash#Gear_fingerprint_and_content-based_chunking_algorithm_FastCDC)
with block sizes around 1MiB - see `chunker.go`

Using zstd compression in speed fastest, which is most suitable for caching use case. See `compressor.go` for other options.

# Building

Build binary - below using static binary that allows to run properly in
environments that are missing some libraries.

```
GOOS=linux GOARCH=amd64 go build -a -ldflags "-extldflags '-static'" github.com/glukasiknuro/bazel-cache-chunking
```

# Running

Below execution will scan all files in a given directory and print stats if
data was chunked and compressed.  Note, that it does not decompress any found
files, so it does not make sense to run it on already compressed data.

```
./bazel-cache-chunking -workers 10 -dir /data/cas
```

# Sample stats

Below is an execution on data from
[bazel-remote](https://github.com/buchgr/bazel-remote) CAS data, from system
that caches builds mainly build with `--compilation_mode=fastbuild`.

```
Processed successfully 468997 files, total size: 1.7 TB

No chunking ZSTD CGO SPEED  size: 343 GB (avg: 732 kB), 20.53% of total  (4.87x effective space)  chunks: 468997    duplicate: 0
        1MB ZSTD CGO SPEED  size: 323 GB (avg: 688 kB), 19.29% of total  (5.18x effective space)  chunks: 1888847   duplicate: 134986
       Gear ZSTD CGO SPEED  size: 244 GB (avg: 519 kB), 14.55% of total  (6.87x effective space)  chunks: 1171361   duplicate: 440797
```
