package main

import (
	"bufio"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Data struct {
	max, min, sum, count int64
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
		line := scanner.Text()
		idx := strings.Index(line, ";")
		if idx == -1 {
			continue
		}
		key := line[:idx]
		temp, err := strconv.ParseFloat(line[idx+1:], 32)
		if err != nil {
			log.Printf("failed to parse float: %v\n", err)
			continue
		}
		v := int64(temp * 10)
		if val, ok := m[key]; !ok {
			d := Data{
				max:   v,
				min:   v,
				sum:   v,
				count: 1,
			}
			m[key] = d
		} else {
			d := Data{
				max:   max(val.max, v),
				min:   min(val.min, v),
				sum:   val.sum + v,
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
