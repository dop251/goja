package goja

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/dop251/goja/ftoa"
	"github.com/dop251/goja/unistring"
)

const hex = "0123456789abcdef"

func (r *Runtime) builtinJSON_parse(call FunctionCall) Value {
	value := r.parseJSON(call.Argument(0).toString().String())

	var reviver func(FunctionCall) Value

	if arg1 := call.Argument(1); arg1 != _undefined {
		reviver, _ = arg1.ToObject(r).self.assertCallable()
	}

	if reviver != nil {
		root := r.NewObject()
		createDataPropertyOrThrow(root, stringEmpty, value)
		return r.builtinJSON_reviveWalk(reviver, root, stringEmpty)
	}

	return value
}

func (r *Runtime) builtinJSON_reviveWalk(reviver func(FunctionCall) Value, holder *Object, name Value) Value {
	value := nilSafe(holder.get(name, nil))

	if object, ok := value.(*Object); ok {
		if isArray(object) {
			length := toLength(object.self.getStr("length", nil))
			for index := int64(0); index < length; index++ {
				name := asciiString(strconv.FormatInt(index, 10))
				value := r.builtinJSON_reviveWalk(reviver, object, name)
				if value == _undefined {
					object.delete(name, false)
				} else {
					createDataProperty(object, name, value)
				}
			}
		} else {
			for _, name := range object.self.stringKeys(false, nil) {
				value := r.builtinJSON_reviveWalk(reviver, object, name)
				if value == _undefined {
					object.self.deleteStr(name.string(), false)
				} else {
					createDataProperty(object, name, value)
				}
			}
		}
	}
	return reviver(FunctionCall{
		This:      holder,
		Arguments: []Value{name, value},
	})
}

type _builtinJSON_stringifyContext struct {
	r                *Runtime
	stack            []*Object
	propertyList     []Value
	replacerFunction func(FunctionCall) Value
	gap, indent      string
	buf              jsonStringifyBuffer
	allAscii         bool
}

func (r *Runtime) builtinJSON_stringify(call FunctionCall) Value {
	ctx := _builtinJSON_stringifyContext{
		r:        r,
		allAscii: true,
	}

	replacer, _ := call.Argument(1).(*Object)
	if replacer != nil {
		if isArray(replacer) {
			length := toLength(replacer.self.getStr("length", nil))
			seen := map[string]bool{}
			propertyList := make([]Value, length)
			length = 0
			for index := range propertyList {
				var name string
				value := replacer.self.getIdx(valueInt(int64(index)), nil)
				switch v := value.(type) {
				case valueFloat, valueInt, String:
					name = value.String()
				case *Object:
					switch v.self.className() {
					case classNumber, classString:
						name = value.String()
					default:
						continue
					}
				default:
					continue
				}
				if seen[name] {
					continue
				}
				seen[name] = true
				propertyList[length] = newStringValue(name)
				length += 1
			}
			ctx.propertyList = propertyList[0:length]
		} else if c, ok := replacer.self.assertCallable(); ok {
			ctx.replacerFunction = c
		}
	}
	if spaceValue := call.Argument(2); spaceValue != _undefined {
		if o, ok := spaceValue.(*Object); ok {
			switch oImpl := o.self.(type) {
			case *primitiveValueObject:
				switch oImpl.pValue.(type) {
				case valueInt, valueFloat:
					spaceValue = o.ToNumber()
				}
			case *stringObject:
				spaceValue = o.ToString()
			}
		}
		isNum := false
		var num int64
		if i, ok := spaceValue.(valueInt); ok {
			num = int64(i)
			isNum = true
		} else if f, ok := spaceValue.(valueFloat); ok {
			num = int64(f)
			isNum = true
		}
		if isNum {
			if num > 0 {
				if num > 10 {
					num = 10
				}
				ctx.gap = strings.Repeat(" ", int(num))
			}
		} else {
			if s, ok := spaceValue.(String); ok {
				str := s.String()
				if len(str) > 10 {
					ctx.gap = str[:10]
				} else {
					ctx.gap = str
				}
			}
		}
	}

	if ctx.do(call.Argument(0)) {
		if ctx.allAscii {
			return asciiString(ctx.buf.String())
		} else {
			return &importedString{
				s: ctx.buf.String(),
			}
		}
	}
	return _undefined
}

func (ctx *_builtinJSON_stringifyContext) do(v Value) bool {
	if ctx.replacerFunction == nil {
		return ctx.strValue(jsonStringifyRawKey(stringEmpty.string()), v, nil)
	}
	holder := ctx.r.NewObject()
	createDataPropertyOrThrow(holder, stringEmpty, v)
	return ctx.str(jsonStringifyRawKey(stringEmpty.string()), holder)
}

type jsonStringifyKey struct {
	raw   unistring.String
	value Value
	index int64
	kind  uint8
}

const (
	jsonStringifyKeyRaw uint8 = iota
	jsonStringifyKeyValue
	jsonStringifyKeyIndex
)

func jsonStringifyRawKey(key unistring.String) jsonStringifyKey {
	return jsonStringifyKey{raw: key, kind: jsonStringifyKeyRaw}
}

func jsonStringifyValueKey(key Value) jsonStringifyKey {
	return jsonStringifyKey{value: key, kind: jsonStringifyKeyValue}
}

func jsonStringifyIndexKey(index int64) jsonStringifyKey {
	return jsonStringifyKey{index: index, kind: jsonStringifyKeyIndex}
}

func (key jsonStringifyKey) toString() String {
	switch key.kind {
	case jsonStringifyKeyValue:
		return key.value.toString()
	case jsonStringifyKeyIndex:
		return asciiString(strconv.FormatInt(key.index, 10))
	default:
		return stringValueFromRaw(key.raw)
	}
}

func (key jsonStringifyKey) get(holder *Object) Value {
	switch key.kind {
	case jsonStringifyKeyValue:
		return holder.get(key.value, nil)
	case jsonStringifyKeyIndex:
		return holder.self.getIdx(valueInt(key.index), nil)
	default:
		return holder.self.getStr(key.raw, nil)
	}
}

func (ctx *_builtinJSON_stringifyContext) str(key jsonStringifyKey, holder *Object) bool {
	return ctx.strValue(key, nilSafe(key.get(holder)), holder)
}

func (ctx *_builtinJSON_stringifyContext) strValue(key jsonStringifyKey, value Value, holder *Object) bool {
	// Reserve headroom for a value to cross the flush point without buffer growth.
	if ctx.buf.Buffer.Len() >= jsonStringifyChunkSize-jsonStringifyChunkSize/8 {
		ctx.buf.flush()
	}
	switch value.(type) {
	case *Object, *valueBigInt:
		if toJSON, ok := ctx.r.getVStr(value, "toJSON").(*Object); ok {
			if c, ok := toJSON.self.assertCallable(); ok {
				value = c(FunctionCall{
					This:      value,
					Arguments: []Value{key.toString()},
				})
			}
		}
	}

	if ctx.replacerFunction != nil {
		value = ctx.replacerFunction(FunctionCall{
			This:      holder,
			Arguments: []Value{key.toString(), value},
		})
	}

	if o, ok := value.(*Object); ok {
		switch o1 := o.self.(type) {
		case *primitiveValueObject:
			switch pValue := o1.pValue.(type) {
			case valueInt, valueFloat:
				value = o.ToNumber()
			default:
				value = pValue
			}
		case *stringObject:
			value = o.toString()
		case *objectGoReflect:
			if o1.toJson != nil {
				value = ctx.r.ToValue(o1.toJson())
			} else if v, ok := o1.origValue.Interface().(json.Marshaler); ok {
				b, err := v.MarshalJSON()
				if err != nil {
					panic(ctx.r.NewGoError(err))
				}
				ctx.buf.Write(b)
				ctx.allAscii = false
				return true
			} else {
				switch o1.className() {
				case classNumber:
					value = o1.val.ordinaryToPrimitiveNumber()
				case classString:
					value = o1.val.ordinaryToPrimitiveString()
				case classBoolean:
					if o.ToInteger() != 0 {
						value = valueTrue
					} else {
						value = valueFalse
					}
				}
				if o1.exportType() == typeBigInt {
					value = o1.val.ordinaryToPrimitiveNumber()
				}
			}
		}
	}

	switch value1 := value.(type) {
	case valueBool:
		if value1 {
			ctx.buf.WriteString("true")
		} else {
			ctx.buf.WriteString("false")
		}
	case String:
		ctx.quote(value1)
	case valueInt:
		var scratch [24]byte
		ctx.buf.Write(strconv.AppendInt(scratch[:0], int64(value1), 10))
	case valueFloat:
		if !math.IsNaN(float64(value1)) && !math.IsInf(float64(value1), 0) {
			var scratch [32]byte
			ctx.buf.Write(ftoa.FToStr(float64(value1), ftoa.ModeStandard, 0, scratch[:0]))
		} else {
			ctx.buf.WriteString("null")
		}
	case valueNull:
		ctx.buf.WriteString("null")
	case *valueBigInt:
		ctx.r.typeErrorResult(true, "Do not know how to serialize a BigInt")
	case *Object:
		if value1.self.className() == classRawJSON {
			ctx.buf.WriteString(value1.self.getStr("rawJSON", nil).String())
			return true
		}
		for _, object := range ctx.stack {
			if value1.SameAs(object) {
				ctx.r.typeErrorResult(true, "Converting circular structure to JSON")
			}
		}
		if _, ok := value1.self.assertCallable(); ok {
			return false
		}
		ctx.stack = append(ctx.stack, value1)
		if isArray(value1) {
			ctx.ja(value1)
		} else {
			ctx.jo(value1)
		}
		ctx.stack = ctx.stack[:len(ctx.stack)-1]
	default:
		return false
	}
	return true
}

func (ctx *_builtinJSON_stringifyContext) ja(array *Object) {
	var stepback string
	if ctx.gap != "" {
		stepback = ctx.indent
		ctx.indent += ctx.gap
	}
	length := toLength(array.self.getStr("length", nil))
	if length == 0 {
		ctx.buf.WriteString("[]")
		if ctx.gap != "" {
			ctx.indent = stepback
		}
		return
	}

	ctx.buf.WriteByte('[')
	var separator string
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(ctx.indent)
		separator = ",\n" + ctx.indent
	} else {
		separator = ","
	}

	for i := int64(0); i < length; i++ {
		if !ctx.str(jsonStringifyIndexKey(i), array) {
			ctx.buf.WriteString("null")
		}
		if i < length-1 {
			ctx.buf.WriteString(separator)
		}
	}
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(stepback)
		ctx.indent = stepback
	}
	ctx.buf.WriteByte(']')
}

func (ctx *_builtinJSON_stringifyContext) jo(object *Object) {
	var stepback string
	if ctx.gap != "" {
		stepback = ctx.indent
		ctx.indent += ctx.gap
	}

	ctx.buf.WriteByte('{')
	mark := ctx.buf.Len()
	var separator string
	if ctx.gap != "" {
		ctx.buf.WriteByte('\n')
		ctx.buf.WriteString(ctx.indent)
		separator = ",\n" + ctx.indent
	} else {
		separator = ","
	}

	empty := true
	writeProperty := func(name unistring.String, key jsonStringifyKey, value Value) {
		off := ctx.buf.Len()
		if !empty {
			ctx.buf.WriteString(separator)
		}
		ctx.quoteRaw(name)
		if ctx.gap != "" {
			ctx.buf.WriteString(": ")
		} else {
			ctx.buf.WriteByte(':')
		}
		if ctx.strValue(key, value, object) {
			if empty {
				empty = false
			}
		} else {
			ctx.buf.Truncate(off)
		}
	}

	if ctx.propertyList == nil {
		if compact, ok := object.self.(*jsonObject); ok {
			shape := compact.shape
			for i := range shape.keys {
				slot := shape.orderedSlot(i)
				name := shape.keys[slot]
				var value Value
				if current, ok := object.self.(*jsonObject); ok && current == compact && current.shape == shape {
					value = current.values[slot]
				} else {
					value = object.self.getStr(name, nil)
				}
				writeProperty(name, jsonStringifyRawKey(name), nilSafe(value))
			}
		} else {
			for _, name := range object.self.stringKeys(false, nil) {
				writeProperty(name.string(), jsonStringifyValueKey(name), nilSafe(object.get(name, nil)))
			}
		}
	} else {
		for _, name := range ctx.propertyList {
			writeProperty(name.string(), jsonStringifyValueKey(name), nilSafe(object.get(name, nil)))
		}
	}

	if empty {
		ctx.buf.Truncate(mark)
		if ctx.gap != "" {
			ctx.indent = stepback
		}
	} else {
		if ctx.gap != "" {
			ctx.buf.WriteByte('\n')
			ctx.buf.WriteString(stepback)
			ctx.indent = stepback
		}
	}
	ctx.buf.WriteByte('}')
}

func (ctx *_builtinJSON_stringifyContext) quoteRaw(raw unistring.String) {
	if utf16Value := raw.AsUtf16(); utf16Value != nil {
		ctx.quoteUnicode(unicodeString(utf16Value))
	} else {
		ctx.quoteAscii(string(raw))
	}
}

func (ctx *_builtinJSON_stringifyContext) quote(str String) {
	if s, us := devirtualizeString(str); us == nil {
		ctx.quoteAscii(string(s))
	} else {
		ctx.quoteUnicode(us)
	}
}

func (ctx *_builtinJSON_stringifyContext) quoteAscii(s string) {
	ctx.buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c != '"' && c != '\\' {
			continue
		}
		ctx.buf.WriteString(s[start:i])
		switch c {
		case '"', '\\':
			ctx.buf.WriteByte('\\')
			ctx.buf.WriteByte(c)
		case 0x08:
			ctx.buf.WriteString(`\b`)
		case 0x09:
			ctx.buf.WriteString(`\t`)
		case 0x0A:
			ctx.buf.WriteString(`\n`)
		case 0x0C:
			ctx.buf.WriteString(`\f`)
		case 0x0D:
			ctx.buf.WriteString(`\r`)
		default:
			ctx.buf.WriteString(`\u00`)
			ctx.buf.WriteByte(hex[c>>4])
			ctx.buf.WriteByte(hex[c&0xF])
		}
		start = i + 1
	}
	ctx.buf.WriteString(s[start:])
	ctx.buf.WriteByte('"')
}

func (ctx *_builtinJSON_stringifyContext) quoteUnicode(s unicodeString) {
	ctx.buf.WriteByte('"')
	units := []uint16(s[1:]) // skip the BOM header
	for i := 0; i < len(units); i++ {
		c := units[i]
		switch c {
		case '"', '\\':
			ctx.buf.WriteByte('\\')
			ctx.buf.WriteByte(byte(c))
		case 0x08:
			ctx.buf.WriteString(`\b`)
		case 0x09:
			ctx.buf.WriteString(`\t`)
		case 0x0A:
			ctx.buf.WriteString(`\n`)
		case 0x0C:
			ctx.buf.WriteString(`\f`)
		case 0x0D:
			ctx.buf.WriteString(`\r`)
		default:
			switch {
			case c < 0x20:
				ctx.buf.WriteString(`\u00`)
				ctx.buf.WriteByte(hex[c>>4])
				ctx.buf.WriteByte(hex[c&0xF])
			case c < utf8.RuneSelf:
				ctx.buf.WriteByte(byte(c))
			case isUTF16FirstSurrogate(c) && i+1 < len(units) && isUTF16SecondSurrogate(units[i+1]):
				ctx.buf.WriteRune(utf16.DecodeRune(rune(c), rune(units[i+1])))
				i++
				ctx.allAscii = false
			case utf16.IsSurrogate(rune(c)):
				ctx.buf.WriteString(`\u`)
				ctx.buf.WriteByte(hex[c>>12])
				ctx.buf.WriteByte(hex[(c>>8)&0xF])
				ctx.buf.WriteByte(hex[(c>>4)&0xF])
				ctx.buf.WriteByte(hex[c&0xF])
			default:
				ctx.buf.WriteRune(rune(c))
				ctx.allAscii = false
			}
		}
	}
	ctx.buf.WriteByte('"')
}

func (r *Runtime) builtinJSON_rawJSON(call FunctionCall) Value {
	arg := call.Argument(0)

	var jsonString String
	switch dd := arg.(type) {
	case String:
		jsonString = dd
		if jsonString.Length() == 0 {
			panic(r.newSyntaxError("\"\" is unacceptable as raw JSON"))
		}
		first := jsonString.CharAt(0)
		last := jsonString.CharAt(jsonString.Length() - 1)
		if first == '{' || first == '[' ||
			first == ' ' || first == '\n' || first == '\r' || first == '\t' ||
			last == ' ' || last == '\n' || last == '\r' || last == '\t' {
			panic(r.newSyntaxError("\"" + arg.String() + "\" is unacceptable as raw JSON"))
		}
		if !json.Valid([]byte(jsonString.String())) {
			panic(r.newSyntaxError("\"" + arg.String() + "\" is not a valid JSON"))
		}
	case valueBool, valueInt, *valueBigInt, valueFloat, valueNull:
		jsonString = dd.toString()
	case *Symbol:
		panic(r.NewTypeError("Cannot convert a Symbol value to a string"))
	default:
		panic(r.newSyntaxError(arg.String() + " is not a valid JSON"))
	}

	o := r.newBaseObject(nil, classRawJSON)
	o._putProp("rawJSON", jsonString, false, true, false)
	o.preventExtensions(true)
	return o.val
}

func (r *Runtime) builtinJSON_isRawJSON(call FunctionCall) Value {
	arg := call.Argument(0)
	if o, ok := arg.(*Object); ok {
		if o.self.className() == classRawJSON {
			return valueTrue
		}
	}
	return valueFalse
}

func (r *Runtime) getJSON() *Object {
	ret := r.global.JSON
	if ret == nil {
		JSON := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret = JSON.val
		r.global.JSON = ret
		JSON._putProp("parse", r.newNativeFunc(r.builtinJSON_parse, "parse", 2), true, false, true)
		JSON._putProp("stringify", r.newNativeFunc(r.builtinJSON_stringify, "stringify", 3), true, false, true)
		JSON._putProp("rawJSON", r.newNativeFunc(r.builtinJSON_rawJSON, "rawJSON", 1), true, false, true)
		JSON._putProp("isRawJSON", r.newNativeFunc(r.builtinJSON_isRawJSON, "isRawJSON", 1), true, false, true)
		JSON._putSym(SymToStringTag, valueProp(asciiString(classJSON), false, false, true))
	}
	return ret
}
