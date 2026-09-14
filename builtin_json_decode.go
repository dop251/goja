package goja

import (
	"encoding/binary"
	"hash/maphash"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dop251/goja/unistring"
)

const maxJSONParseDepth = 10000

const maxJSONShapes = 4096

type jsonPair struct {
	key   unistring.String
	value Value
}

type jsonParser struct {
	r     *Runtime
	data  string
	pos   int
	depth int
	keys  map[string]unistring.String

	shapeSeed  maphash.Seed
	shapes     map[uint64]*jsonShape
	shapeCount int

	// reusable scratch stacks shared across nesting levels
	pairs    []jsonPair
	elements []Value
}

func (r *Runtime) parseJSON(data string) Value {
	p := jsonParser{r: r, data: data}
	v := p.parseValue()
	p.skipWS()
	if p.pos < len(p.data) {
		p.errUnexpectedToken(p.pos)
	}
	p.clearShapeLinks()
	return v
}

func (p *jsonParser) error(msg string) {
	panic(p.r.newError(p.r.getSyntaxError(), msg))
}

func (p *jsonParser) errUnexpectedEnd() {
	p.error("Unexpected end of JSON input")
}

func (p *jsonParser) errUnexpectedToken(pos int) {
	p.error("Unexpected token " + string(p.data[pos]) + " in JSON at position " + strconv.Itoa(pos))
}

func (p *jsonParser) skipWS() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *jsonParser) parseValue() Value {
	p.skipWS()
	if p.pos >= len(p.data) {
		p.errUnexpectedEnd()
	}
	switch c := p.data[p.pos]; c {
	case '{':
		p.pos++
		return p.parseObject()
	case '[':
		p.pos++
		return p.parseArray()
	case '"':
		p.pos++
		return p.parseString()
	case 't':
		p.parseLiteral("true")
		return valueTrue
	case 'f':
		p.parseLiteral("false")
		return valueFalse
	case 'n':
		p.parseLiteral("null")
		return _null
	default:
		if c == '-' || c >= '0' && c <= '9' {
			return p.parseNumber()
		}
		p.errUnexpectedToken(p.pos)
	}
	return nil
}

func (p *jsonParser) parseLiteral(lit string) {
	if len(p.data)-p.pos < len(lit) {
		p.errUnexpectedEnd()
	}
	if p.data[p.pos:p.pos+len(lit)] != lit {
		p.errUnexpectedToken(p.pos)
	}
	p.pos += len(lit)
}

func (p *jsonParser) enter() {
	p.depth++
	if p.depth > maxJSONParseDepth {
		p.error("Maximum JSON parse depth exceeded")
	}
}

func (p *jsonParser) parseObject() *Object {
	p.enter()
	p.skipWS()
	if p.pos >= len(p.data) {
		p.errUnexpectedEnd()
	}
	if p.data[p.pos] == '}' {
		p.pos++
		p.depth--
		return p.r.NewObject()
	}
	base := len(p.pairs)
	for {
		p.skipWS()
		if p.pos >= len(p.data) {
			p.errUnexpectedEnd()
		}
		if p.data[p.pos] != '"' {
			p.errUnexpectedToken(p.pos)
		}
		p.pos++
		key := p.parseObjectKey()
		p.skipWS()
		if p.pos >= len(p.data) {
			p.errUnexpectedEnd()
		}
		if p.data[p.pos] != ':' {
			p.errUnexpectedToken(p.pos)
		}
		p.pos++
		value := p.parseValue()
		p.pairs = append(p.pairs, jsonPair{key: key, value: value})
		p.skipWS()
		if p.pos >= len(p.data) {
			p.errUnexpectedEnd()
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case '}':
			p.pos++
			p.depth--
			object := p.makeObject(p.pairs[base:])
			p.pairs = p.pairs[:base]
			return object
		default:
			p.errUnexpectedToken(p.pos)
		}
	}
}

func compactJSONPairs(pairs []jsonPair) []jsonPair {
	unique := pairs[:0]
	if len(pairs) <= 16 {
		for _, pair := range pairs {
			found := false
			for i := range unique {
				if unique[i].key == pair.key {
					unique[i].value = pair.value
					found = true
					break
				}
			}
			if !found {
				unique = append(unique, pair)
			}
		}
		return unique
	}

	indices := make(map[unistring.String]int, len(pairs))
	for _, pair := range pairs {
		if index, exists := indices[pair.key]; exists {
			unique[index].value = pair.value
		} else {
			indices[pair.key] = len(unique)
			unique = append(unique, pair)
		}
	}
	return unique
}

func (p *jsonParser) shapeFor(pairs []jsonPair) *jsonShape {
	if p.shapes == nil {
		p.shapeSeed = maphash.MakeSeed()
		p.shapes = make(map[uint64]*jsonShape)
	}

	var hasher maphash.Hash
	hasher.SetSeed(p.shapeSeed)
	var size [4]byte
	for _, pair := range pairs {
		binary.LittleEndian.PutUint32(size[:], uint32(len(pair.key)))
		_, _ = hasher.Write(size[:])
		_, _ = hasher.WriteString(string(pair.key))
	}
	hash := hasher.Sum64()
	for shape := p.shapes[hash]; shape != nil; shape = shape.hashNext {
		if len(shape.keys) != len(pairs) {
			continue
		}
		match := true
		for i, pair := range pairs {
			if shape.keys[i] != pair.key {
				match = false
				break
			}
		}
		if match {
			return shape
		}
	}

	if p.shapeCount >= maxJSONShapes {
		return nil
	}
	keys := make([]unistring.String, len(pairs))
	for i, pair := range pairs {
		keys[i] = pair.key
	}
	shape := newJSONShape(keys)
	shape.hashNext = p.shapes[hash]
	p.shapes[hash] = shape
	p.shapeCount++
	return shape
}

func (p *jsonParser) clearShapeLinks() {
	for _, head := range p.shapes {
		for shape := head; shape != nil; {
			next := shape.hashNext
			shape.hashNext = nil
			shape = next
		}
	}
}

// makeObject builds a shape-backed plain object. If the per-parse cache has
// reached its bound, previously unseen layouts use the ordinary representation.
func (p *jsonParser) makeObject(pairs []jsonPair) *Object {
	pairs = compactJSONPairs(pairs)
	if shape := p.shapeFor(pairs); shape != nil {
		values := make([]Value, len(pairs))
		for i, pair := range pairs {
			values[i] = pair.value
		}
		return p.r.newJSONObject(shape, values)
	}

	object := &Object{runtime: p.r}
	mapSize := 0
	if len(pairs) > 8 {
		mapSize = len(pairs)
	}
	self := &baseObject{
		class:      classObject,
		val:        object,
		prototype:  p.r.global.ObjectPrototype,
		extensible: true,
		values:     make(map[unistring.String]Value, mapSize),
		propNames:  make([]unistring.String, 0, len(pairs)),
	}
	object.self = self
	for i := range pairs {
		self._putProp(pairs[i].key, pairs[i].value, true, true, true)
	}
	return object
}

func (p *jsonParser) parseArray() *Object {
	p.enter()
	p.skipWS()
	if p.pos >= len(p.data) {
		p.errUnexpectedEnd()
	}
	if p.data[p.pos] == ']' {
		p.pos++
		p.depth--
		return p.r.newArrayValues(nil)
	}
	base := len(p.elements)
	for {
		p.elements = append(p.elements, p.parseValue())
		p.skipWS()
		if p.pos >= len(p.data) {
			p.errUnexpectedEnd()
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case ']':
			p.pos++
			p.depth--
			if p.depth == 0 && cap(p.elements)-len(p.elements) <= len(p.elements)/8 {
				// No outer array needs the scratch storage after the root closes.
				// Reuse only tightly sized buffers to limit retained spare capacity.
				values := p.elements
				clear(values[len(values):cap(values)])
				p.elements = nil
				return p.r.newArrayValues(values)
			}
			values := make([]Value, len(p.elements)-base)
			copy(values, p.elements[base:])
			p.elements = p.elements[:base]
			return p.r.newArrayValues(values)
		default:
			p.errUnexpectedToken(p.pos)
		}
	}
}

// scanString finds the end of the string literal starting at p.pos (after the
// opening quote) without decoding escapes.
func (p *jsonParser) scanString() (end int, hasEscapes, hasNonAscii bool) {
	for i := p.pos; i < len(p.data); i++ {
		switch c := p.data[i]; {
		case c == '"':
			return i, hasEscapes, hasNonAscii
		case c == '\\':
			hasEscapes = true
			i++ // skip the escaped char; it cannot be a closing quote
		case c < 0x20:
			p.errUnexpectedToken(i)
		case c >= utf8.RuneSelf:
			hasNonAscii = true
		}
	}
	p.errUnexpectedEnd()
	return
}

// parseObjectKey parses a string literal at p.pos (after the opening quote),
// interning the result so repeated keys are only allocated once.
func (p *jsonParser) parseObjectKey() unistring.String {
	end, hasEscapes, hasNonAscii := p.scanString()
	raw := p.data[p.pos:end]
	if key, ok := p.keys[raw]; ok {
		p.pos = end + 1
		return key
	}
	var key unistring.String
	var mapKey string
	if !hasEscapes && !hasNonAscii {
		mapKey = strings.Clone(raw)
		key = unistring.String(mapKey)
		p.pos = end + 1
	} else {
		mapKey = strings.Clone(raw)
		key = p.parseString().string()
	}
	if p.keys == nil {
		p.keys = make(map[string]unistring.String)
	}
	p.keys[mapKey] = key
	return key
}

// parseString parses a string literal at p.pos (after the opening quote).
func (p *jsonParser) parseString() String {
	end, hasEscapes, hasNonAscii := p.scanString()
	if !hasEscapes {
		raw := p.data[p.pos:end]
		p.pos = end + 1
		if !hasNonAscii {
			return asciiString(strings.Clone(raw))
		}
		return stringValueFromRaw(unistring.NewFromString(raw))
	}
	return p.parseStringSlow(end)
}

func (p *jsonParser) parseStringSlow(approxEnd int) String {
	var sb StringBuilder
	sb.Grow(approxEnd - p.pos)
	i := p.pos
	start := i
	for i < len(p.data) {
		switch c := p.data[i]; {
		case c == '"':
			sb.WriteUTF8String(p.data[start:i])
			p.pos = i + 1
			return sb.String()
		case c == '\\':
			sb.WriteUTF8String(p.data[start:i])
			i++
			if i >= len(p.data) {
				p.errUnexpectedEnd()
			}
			switch e := p.data[i]; e {
			case '"', '\\', '/':
				sb.WriteRune(rune(e))
				i++
			case 'b':
				sb.WriteRune('\b')
				i++
			case 'f':
				sb.WriteRune('\f')
				i++
			case 'n':
				sb.WriteRune('\n')
				i++
			case 'r':
				sb.WriteRune('\r')
				i++
			case 't':
				sb.WriteRune('\t')
				i++
			case 'u':
				if i+4 >= len(p.data) {
					p.errUnexpectedEnd()
				}
				var code rune
				for j := i + 1; j <= i+4; j++ {
					code <<= 4
					switch d := p.data[j]; {
					case d >= '0' && d <= '9':
						code |= rune(d - '0')
					case d >= 'a' && d <= 'f':
						code |= rune(d-'a') + 10
					case d >= 'A' && d <= 'F':
						code |= rune(d-'A') + 10
					default:
						p.errUnexpectedToken(j)
					}
				}
				// written as a raw UTF-16 unit; lone surrogates are preserved
				sb.WriteRune(code)
				i += 5
			default:
				p.errUnexpectedToken(i)
			}
			start = i
		case c < 0x20:
			p.errUnexpectedToken(i)
		default:
			i++
		}
	}
	p.errUnexpectedEnd()
	return nil
}

func (p *jsonParser) parseNumber() Value {
	start := p.pos
	i := p.pos
	neg := false
	if p.data[i] == '-' {
		neg = true
		i++
	}
	if i >= len(p.data) {
		p.errUnexpectedEnd()
	}
	var intVal int64
	digits := 0
	if p.data[i] == '0' {
		i++
		digits = 1
	} else if c := p.data[i]; c >= '1' && c <= '9' {
		for i < len(p.data) {
			c := p.data[i]
			if c < '0' || c > '9' {
				break
			}
			intVal = intVal*10 + int64(c-'0')
			digits++
			i++
		}
	} else {
		p.errUnexpectedToken(i)
	}
	isFloat := false
	if i < len(p.data) && p.data[i] == '.' {
		isFloat = true
		i++
		if i >= len(p.data) || p.data[i] < '0' || p.data[i] > '9' {
			p.errNumber(i)
		}
		for i < len(p.data) && p.data[i] >= '0' && p.data[i] <= '9' {
			i++
		}
	}
	if i < len(p.data) && (p.data[i] == 'e' || p.data[i] == 'E') {
		isFloat = true
		i++
		if i < len(p.data) && (p.data[i] == '+' || p.data[i] == '-') {
			i++
		}
		if i >= len(p.data) || p.data[i] < '0' || p.data[i] > '9' {
			p.errNumber(i)
		}
		for i < len(p.data) && p.data[i] >= '0' && p.data[i] <= '9' {
			i++
		}
	}
	p.pos = i
	// 15 digits always fit a float64 exactly, so the int fast path matches ToNumber semantics
	if !isFloat && digits <= 15 {
		if neg {
			if intVal == 0 {
				return _negativeZero
			}
			intVal = -intVal
		}
		return intToValue(intVal)
	}
	f, err := strconv.ParseFloat(p.data[start:i], 64)
	if err != nil {
		if numErr, ok := err.(*strconv.NumError); !ok || numErr.Err != strconv.ErrRange {
			p.errNumber(start)
		}
		// out-of-range values round to ±Inf per ToNumber semantics
	}
	return floatToValue(f)
}

func (p *jsonParser) errNumber(pos int) {
	if pos >= len(p.data) {
		p.errUnexpectedEnd()
	}
	p.errUnexpectedToken(pos)
}
