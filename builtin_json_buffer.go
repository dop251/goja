package goja

import (
	"bytes"
	"strings"
)

const jsonStringifyChunkSize = 64 * 1024

// jsonStringifyBuffer avoids repeatedly copying large output prefixes during
// growth. The serializer moves chunks aside between values; writes use
// bytes.Buffer directly, and String joins chunks into one final allocation.
type jsonStringifyBuffer struct {
	bytes.Buffer
	chunks [][]byte
	size   int
}

func (b *jsonStringifyBuffer) flush() {
	b.chunks = append(b.chunks, b.Buffer.Bytes())
	b.size += b.Buffer.Len()
	b.Buffer = bytes.Buffer{}
	b.Buffer.Grow(jsonStringifyChunkSize)
}

func (b *jsonStringifyBuffer) Len() int {
	return b.size + b.Buffer.Len()
}

func (b *jsonStringifyBuffer) Truncate(n int) {
	if n < 0 || n > b.Len() {
		panic("jsonStringifyBuffer: invalid truncation")
	}
	for n < b.size {
		last := len(b.chunks) - 1
		chunk := b.chunks[last]
		b.chunks[last] = nil
		b.chunks = b.chunks[:last]
		b.size -= len(chunk)
		b.Buffer = *bytes.NewBuffer(chunk)
	}
	b.Buffer.Truncate(n - b.size)
}

func (b *jsonStringifyBuffer) String() string {
	if len(b.chunks) == 0 {
		return b.Buffer.String()
	}
	var result strings.Builder
	result.Grow(b.Len())
	for _, chunk := range b.chunks {
		result.Write(chunk)
	}
	result.Write(b.Buffer.Bytes())
	return result.String()
}

func (b *jsonStringifyBuffer) Bytes() []byte {
	if len(b.chunks) == 0 {
		return b.Buffer.Bytes()
	}
	result := make([]byte, 0, b.Len())
	for _, chunk := range b.chunks {
		result = append(result, chunk...)
	}
	return append(result, b.Buffer.Bytes()...)
}
