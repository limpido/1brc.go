package main

import (
	"bytes"
	"fmt"
	"log"
	"maps"
	"os"
	"runtime/pprof"
	"slices"
)

type Data struct {
	max, min, sum, count int64
}

func parseFloat(num []byte) int64 {
	negative := false
	var n int64
	for _, c := range num {
		if c == '-' {
			negative = true
			continue
		}
		if c == '.' {
			continue
		}
		n = n*10 + int64(c-'0')
	}
	if negative {
		return -n
	}
	return n
}

func processChunk(f *os.File, startOffset, chunkSize int64, ch chan map[string]Data) {
	buf := make([]byte, chunkSize)
	n, err := f.ReadAt(buf, startOffset)
	if err != nil {
		log.Fatalf("error reading file: %v\n", err)
		return
	}

	m := make(map[string]Data)
	l := 0
	if startOffset > 0 {
		// start after the first '\n'
		idx := bytes.IndexByte(buf, '\n')
		if idx != -1 {
			l = idx + 1
		}
	}
	for r := l; r < n; r++ {
		if buf[r] != '\n' {
			continue
		}
		idx := bytes.IndexByte(buf[l:r], ';')
		if idx == -1 {
			continue
		}
		key := string(buf[l : idx+l])
		temp := parseFloat(buf[idx+l+1 : r])
		if val, ok := m[key]; !ok {
			d := Data{
				max:   temp,
				min:   temp,
				sum:   temp,
				count: 1,
			}
			m[key] = d
		} else {
			d := Data{
				max:   max(val.max, temp),
				min:   min(val.min, temp),
				sum:   val.sum + temp,
				count: val.count + 1,
			}
			m[key] = d
		}
		l = r + 1
	}

	// collect leftover bytes
	var leftover []byte
	if l < len(buf) {
		leftover = buf[l:]
	}

	// keep reading until hit first '\n', combine with the leftover bytes
	tmpBuf := make([]byte, 128)
	curPos := startOffset + chunkSize
	for {
		n, err := f.ReadAt(tmpBuf, curPos)
		if n == 0 || err != nil {
			break
		}
		idx := bytes.IndexByte(tmpBuf[:n], '\n')
		if idx == -1 {
			leftover = append(leftover, tmpBuf...)
			curPos += int64(n)
			continue
		} else {
			leftover = append(leftover, tmpBuf[:idx]...)
			break
		}
	}

	if len(leftover) > 0 {
		idx := bytes.IndexByte(leftover, ';')
		if idx != -1 {
			key := string(leftover[:idx])
			temp := parseFloat(leftover[idx+1:])
			if val, ok := m[key]; !ok {
				m[key] = Data{max: temp, min: temp, sum: temp, count: 1}
			} else {
				m[key] = Data{max: max(val.max, temp), min: min(val.min, temp), sum: val.sum + temp, count: val.count + 1}
			}
		}
	}

	ch <- m
}

func main() {
	prof, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(prof)
	defer pprof.StopCPUProfile()

	filename := "measurements.txt"
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fileInfo, err := os.Stat(filename)
	if err != nil {
		log.Fatalf("Error getting file info: %v\n", err)
		os.Exit(1)
	}

	m := map[string]Data{}

	// divide file into chunks
	// process concurrently with goroutines
	numWorkers := 8
	ch := make(chan map[string]Data)
	defer close(ch)
	chunkSize := fileInfo.Size() / int64(numWorkers)
	for i := 0; i < numWorkers; i++ {
		startOffset := int64(i) * chunkSize
		go processChunk(f, startOffset, chunkSize, ch)
	}

	// read result from the channel
	for partialM := range ch {
		numWorkers--
		// merge maps
		for k, v := range partialM {
			if cur, ok := m[k]; !ok {
				m[k] = v
			} else {
				m[k] = Data{
					max:   max(cur.max, v.max),
					min:   min(cur.min, v.min),
					sum:   cur.sum + v.sum,
					count: cur.count + v.count,
				}
			}
		}
		if numWorkers == 0 {
			break
		}
	}

	// write to output file
	outF, err := os.Create("output.txt")
	if err != nil {
		log.Fatalf("failed to create output file: %v\n", err)
	}
	defer outF.Close()
	for _, k := range slices.Sorted(maps.Keys(m)) {
		v := m[k]
		avg := float64(v.sum) / float64(v.count) / 10.0
		line := fmt.Sprintf("%s;%.1f;%.1f;%.1f\n", k, float64(v.max)/10.0, float64(v.min)/10.0, avg)
		_, err := outF.WriteString(line)
		if err != nil {
			log.Fatalf("failed to write to output file: %v\n", err)
		}
	}
}
