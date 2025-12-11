package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"maps"
	"os"
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

func main() {
	m := map[string]Data{}
	filename := "measurements.txt"
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file: %v\n", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		idx := bytes.IndexByte(line, ';')
		if idx == -1 {
			continue
		}
		key := string(line[:idx])
		temp := parseFloat(line[idx+1:])
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
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("scanner error: %v\n", err)
	}

	// write to output file
	outF, err := os.Create("output.txt")
	if err != nil {
		log.Fatalf("failed to create output file: %v\n", err)
	}
	defer outF.Close()
	for _, k := range slices.Sorted(maps.Keys(m)) {
		v := m[k]
		avg := float32(v.sum) / float32(v.count) / 10.0
		line := fmt.Sprintf("%s;%.1f;%.1f;%.1f\n", k, float32(v.max)/10.0, float32(v.min)/10.0, avg)
		_, err := outF.WriteString(line)
		if err != nil {
			log.Fatalf("failed to write to output file: %v\n", err)
		}
	}
}
