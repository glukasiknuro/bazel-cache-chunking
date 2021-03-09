package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)
import "flag"

var rootDir = flag.String("dir", ".", "Root directory to scan for files")
var workers = flag.Int("workers", 0, "Number of concurrent workers - 0 for num cpus")
var maxFiles = flag.Int("max_files", 0, "Maximum number of files to process")

type FileData struct {
	Path string
	Size int64
}

func getFiles() []FileData {
	var files []FileData
	err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Ignoring error for: %s, err: %s\n", path, err)
			return nil
		}
		if info.Mode().IsRegular() {
			files = append(files, FileData{Path: path, Size: info.Size()})
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

func runProcessorsForFile(f FileData, processors []*Processor) bool {
	in, err := os.OpenFile(f.Path, os.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("Unable to open file, ignoring: %s\n", f.Path)
		return false
	}

	var w []io.Writer
	for _, p := range processors {
		wc := p.NewWriter()
		defer wc.Close()
		w = append(w, wc)
	}

	_, err = io.Copy(io.MultiWriter(w...), in)
	if err != nil {
		fmt.Printf("Unable to process file: %s, err: %s.\n", f.Path, err)
		return false
	}
	return true
}

func runProcessors(files []FileData, processors []*Processor) (int, int64) {
	var lastUpdate time.Time
	var currIdx, processedFiles, processedSize int64

	jobs := *workers
	if jobs == 0 {
		jobs = runtime.NumCPU()
	}
	fmt.Printf("Using %d workers\n", jobs)

	toProcess := len(files)
	if *maxFiles > 0 && *maxFiles < toProcess {
		toProcess = *maxFiles
	}

	var wg sync.WaitGroup
	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go func() {
			for {
				idx := atomic.AddInt64(&currIdx, 1) - 1
				if idx >= int64(toProcess) {
					wg.Done()
					return
				}
				f := files[idx]

				if lastUpdate.Add(time.Second).Before(time.Now()) {
					fmt.Printf("[%d/%d] Processing %s\n", processedFiles+1, toProcess, f.Path)
					lastUpdate = time.Now()
				}

				if runProcessorsForFile(f, processors) {
					atomic.AddInt64(&processedFiles, 1)
					atomic.AddInt64(&processedSize, f.Size)
				}
			}
		}()
	}
	wg.Wait()

	return int(processedFiles), processedSize
}

func processFiles(files []FileData) {
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	fmt.Printf(
		"Got %s files, total size: %s (avg: %s)\n",
		humanize.Comma(int64(len(files))),
		humanize.Bytes(uint64(totalSize)),
		humanize.Bytes(uint64(totalSize)/uint64(len(files))))

	processors := []*Processor{
		//{
		//	ChunkerType: NoChunk,
		//	Compressor:  Identity,
		//},
		{
			ChunkerType: NoChunk,
			Compressor:  ZstdCgoSpeed,
		},
		{
			ChunkerType: OneMb,
			Compressor:  ZstdCgoSpeed,
		},
		{
			ChunkerType: Gear,
			Compressor:  ZstdCgoSpeed,
		},
		//{
		//	ChunkerType: OneMb,
		//	Compressor:  Gzip,
		//},
	}

	num, processedSize := runProcessors(files, processors)

	fmt.Printf("Processed successfully %d files, total size: %s\n", num, humanize.Bytes(uint64(processedSize)))
	fmt.Println()

	for _, p := range processors {
		fmt.Printf("%20s %20s", p.ChunkerType, p.Compressor)

		uniqueSize := p.GetSize()
		ratio := float64(uniqueSize) / float64(processedSize)

		fmt.Printf(
			"  size: %10s (avg: %10s),  %6.2f%% of total  (%5.2fx effective space)  %s",
			humanize.Bytes(uint64(uniqueSize)),
			humanize.Bytes(uint64(uniqueSize)/uint64(num)),
			ratio*100,
			1/ratio,
			p.GetExtraStats())

		fmt.Println()
	}
}

func main() {
	flag.Parse()

	root, err := filepath.Abs(*rootDir)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Scanning files in: %s\n", root)

	files := getFiles()
	if len(files) == 0 {
		fmt.Printf("No files found")
		return
	}
	processFiles(files)
}
