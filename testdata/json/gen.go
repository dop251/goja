// Command gen generates deterministic JSON benchmark fixtures.
// Run from the repo root: go run ./testdata/json/gen.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// lcg is a fixed-seed linear congruential generator so output is reproducible.
type lcg struct{ state uint64 }

func (l *lcg) next() uint64 {
	l.state = l.state*6364136223846793005 + 1442695040888963407
	return l.state >> 11
}

func (l *lcg) intn(n int) int { return int(l.next() % uint64(n)) }

func (l *lcg) float() float64 { return float64(l.next()%1_000_000_00) / 100.0 }

var asciiWords = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey",
	"xray", "yankee", "zulu", "server", "client", "request", "response",
}

var unicodeWords = []string{
	"héllo wörld", "こんにちは世界", "Привет мир", "مرحبا بالعالم", "שלום עולם",
	"你好，世界", "안녕하세요", "γειά σου κόσμε", "🚀 launch", "café ☕", "naïve résumé",
	"Zürich", "São Paulo", "Kraków", "İstanbul", "東京タワー", "emoji 😀😃😄",
}

func (l *lcg) words(dict []string, n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(dict[l.intn(len(dict))])
	}
	return buf.String()
}

func genMedium(l *lcg, count int) interface{} {
	arr := make([]interface{}, count)
	for i := range arr {
		friends := make([]interface{}, 3)
		for j := range friends {
			friends[j] = map[string]interface{}{
				"id":   i*10 + j,
				"name": l.words(asciiWords, 2),
			}
		}
		tags := make([]interface{}, 5)
		for j := range tags {
			tags[j] = asciiWords[l.intn(len(asciiWords))]
		}
		arr[i] = map[string]interface{}{
			"id":        1000000 + i,
			"index":     i,
			"guid":      fmt.Sprintf("%08x-%04x-%04x-%08x", l.next()&0xffffffff, l.next()&0xffff, l.next()&0xffff, l.next()&0xffffffff),
			"isActive":  l.intn(2) == 0,
			"balance":   l.float(),
			"name":      l.words(asciiWords, 3),
			"about":     l.words(asciiWords, 12),
			"latitude":  l.float()/10000 - 90,
			"longitude": l.float()/10000 - 90,
			"tags":      tags,
			"friends":   friends,
		}
	}
	return arr
}

func genUnicode(l *lcg) interface{} {
	arr := make([]interface{}, 400)
	for i := range arr {
		arr[i] = map[string]interface{}{
			"id":    i,
			"title": l.words(unicodeWords, 3),
			"body":  l.words(unicodeWords, 10),
			"note":  "line1\nline2\t\"quoted\" \\backslash",
		}
	}
	return arr
}

func genNumbers(l *lcg) interface{} {
	arr := make([]interface{}, 5000)
	for i := range arr {
		switch i % 4 {
		case 0:
			arr[i] = int64(l.next() % 1_000_000)
		case 1:
			arr[i] = -int64(l.next() % 100_000)
		case 2:
			arr[i] = l.float()
		default:
			arr[i] = l.float() / 1e6
		}
	}
	return arr
}

func genStrings(l *lcg) interface{} {
	arr := make([]interface{}, 2000)
	for i := range arr {
		s := l.words(asciiWords, 1+l.intn(15))
		if i%17 == 0 {
			s += " \"escaped\" \\ \n end"
		}
		arr[i] = s
	}
	return arr
}

func main() {
	dir := filepath.Join("testdata", "json")
	fixtures := map[string]func(*lcg) interface{}{
		"medium.json":      func(l *lcg) interface{} { return genMedium(l, 500) },
		"medium-10x.json":  func(l *lcg) interface{} { return genMedium(l, 5000) },
		"medium-100x.json": func(l *lcg) interface{} { return genMedium(l, 50000) },
		"unicode.json":     genUnicode,
		"numbers.json":     genNumbers,
		"strings.json":     genStrings,
	}
	for name, gen := range fixtures {
		l := &lcg{state: 42}
		data, err := json.Marshal(gen(l))
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			panic(err)
		}
		fmt.Printf("%s: %d bytes\n", name, len(data))
	}
}
