package main

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"runtime/pprof"
	"slices"
)

const (
	HashTableSize = 2048
)

type HashTableEntry struct {
	key []byte
	val *Data
}

type HashTable struct {
	size uint64
	arr  []HashTableEntry
}

func hashBytes(data []byte) uint64 {
	hasher := fnv.New64a()
	hasher.Write(data)
	return hasher.Sum64()
}

func (ht *HashTable) init() {
	ht.size = HashTableSize
	ht.arr = make([]HashTableEntry, HashTableSize)
}

func (ht *HashTable) insert(key []byte, val *Data) {
	idx := hashBytes(key) % ht.size
	i := idx
	for {
		if ht.arr[i].key != nil {
			i++
			if i == ht.size {
				i = 0
			}
			continue
		}
		copyKey := make([]byte, len(key))
		copy(copyKey, key)
		ht.arr[i] = HashTableEntry{copyKey, val}
		break
	}
}

func (ht *HashTable) get(key []byte) (*Data, bool) {
	idx := hashBytes(key) % ht.size
	i := idx
	for ht.arr[i].key != nil {
		if bytes.Equal(key, ht.arr[i].key) {
			return ht.arr[i].val, true
		}
		i++
		if i == ht.size {
			i = 0
		}
	}
	return nil, false
}

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

func processChunk(f *os.File, startOffset, chunkSize int64, ch chan *HashTable) {
	buf := make([]byte, chunkSize)
	n, err := f.ReadAt(buf, startOffset)
	if err != nil {
		log.Fatalf("error reading file: %v\n", err)
		return
	}

	m := new(HashTable)
	m.init()
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
		key := buf[l : idx+l]
		temp := parseFloat(buf[idx+l+1 : r])
		if cur, ok := m.get(key); !ok {
			m.insert(key, &Data{
				max:   temp,
				min:   temp,
				sum:   temp,
				count: 1,
			})
		} else {
			cur.max = max(cur.max, temp)
			cur.min = min(cur.min, temp)
			cur.sum += temp
			cur.count++
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
			key := leftover[:idx]
			temp := parseFloat(leftover[idx+1:])
			if cur, ok := m.get(key); !ok {
				m.insert(key, &Data{
					max:   temp,
					min:   temp,
					sum:   temp,
					count: 1,
				})
			} else {
				cur.max = max(cur.max, temp)
				cur.min = min(cur.min, temp)
				cur.sum += temp
				cur.count++
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

	m := new(HashTable)
	m.init()

	// divide file into chunks
	// process concurrently with goroutines
	numWorkers := 8
	ch := make(chan *HashTable)
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
		for _, htEntry := range partialM.arr {
			if htEntry.key == nil {
				continue
			}
			if cur, ok := m.get(htEntry.key); !ok {
				m.insert(htEntry.key, htEntry.val)
			} else {
				cur.max = max(cur.max, htEntry.val.max)
				cur.min = min(cur.min, htEntry.val.min)
				cur.sum += htEntry.val.sum
				cur.count += htEntry.val.count
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

	// sort by key
	slices.SortFunc(m.arr, func(e1, e2 HashTableEntry) int {
		return bytes.Compare(e1.key, e2.key)
	})

	for _, entry := range m.arr {
		if entry.key == nil {
			continue
		}
		avg := float64(entry.val.sum) / float64(entry.val.count) / 10.0
		line := fmt.Sprintf("%s;%.1f;%.1f;%.1f\n", string(entry.key),
			float64(entry.val.max)/10.0, float64(entry.val.min)/10.0, avg)
		_, err := outF.WriteString(line)
		if err != nil {
			log.Fatalf("failed to write to output file: %v\n", err)
		}
	}
}
