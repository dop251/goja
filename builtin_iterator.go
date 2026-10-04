package goja

import (
	"math"

	"github.com/dop251/goja/unistring"
)

type iteratorHelperState uint8

const (
	iteratorHelperSuspendedStart iteratorHelperState = iota
	iteratorHelperSuspendedYield
	iteratorHelperExecuting
	iteratorHelperCompleted
)

// iteratorHelperObject is the object returned by the lazy Iterator.prototype methods (map, filter, take, drop
// and flatMap). It behaves like a generator: step produces the next value, or reports that the helper is done.
type iteratorHelperObject struct {
	baseObject
	underlying *iteratorRecord
	step       func() (value Value, ok bool)
	// abort, if set, replaces the default closing of the underlying iterator in return().
	abort func()
	state iteratorHelperState
}

func (h *iteratorHelperObject) nextResult(_ Value) (Value, bool) {
	r := h.val.runtime
	switch h.state {
	case iteratorHelperExecuting:
		panic(r.NewTypeError("Iterator Helper is already running"))
	case iteratorHelperCompleted:
		return _undefined, false
	}
	h.state = iteratorHelperExecuting
	defer func() {
		if h.state == iteratorHelperExecuting {
			h.state = iteratorHelperCompleted
		}
	}()
	value, ok := h.step()
	if !ok {
		h.state = iteratorHelperCompleted
		return _undefined, false
	}
	h.state = iteratorHelperSuspendedYield
	return value, true
}

func (h *iteratorHelperObject) next() Value {
	value, valid := h.nextResult(nil)
	return h.val.runtime.createIterResultObject(value, !valid)
}

func (h *iteratorHelperObject) _return() Value {
	r := h.val.runtime
	switch h.state {
	case iteratorHelperExecuting:
		panic(r.NewTypeError("Iterator Helper is already running"))
	case iteratorHelperCompleted:
		return r.createIterResultObject(_undefined, true)
	case iteratorHelperSuspendedStart:
		h.state = iteratorHelperCompleted
		h.underlying.returnIter()
		return r.createIterResultObject(_undefined, true)
	}
	h.state = iteratorHelperExecuting
	defer func() {
		h.state = iteratorHelperCompleted
	}()
	if h.abort != nil {
		h.abort()
	} else {
		h.underlying.returnIter()
	}
	return r.createIterResultObject(_undefined, true)
}

// wrapForValidIteratorObject is the object returned by Iterator.from() for iterators that do not inherit from
// Iterator.prototype.
type wrapForValidIteratorObject struct {
	baseObject
	iterated *iteratorRecord
}

func (r *Runtime) newIteratorHelper(underlying *iteratorRecord, step func() (Value, bool)) *iteratorHelperObject {
	h := &iteratorHelperObject{
		underlying: underlying,
		step:       step,
	}
	o := &Object{runtime: r}
	h.class = classObject
	h.val = o
	h.extensible = true
	o.self = h
	h.prototype = r.getIteratorHelperPrototype()
	h.init()
	return h
}

func (r *Runtime) getIteratorDirect(obj *Object) *iteratorRecord {
	var next func(FunctionCall) Value
	var nextRes func(Value) (Value, bool)
	if nextObj, ok := obj.self.getStr("next", nil).(*Object); ok {
		if call, ok := nextObj.self.assertCallable(); ok {
			next = call
			if nf, ok := nextObj.self.(*iteratorNextFunction); ok {
				nextRes = nf.getNextResult(obj)
			}
		}
	}
	return &iteratorRecord{
		iterator: obj,
		next:     next,
		nextRes:  nextRes,
	}
}

func (r *Runtime) getIteratorFlattenable(obj Value, iterateStrings bool) *iteratorRecord {
	if _, ok := obj.(*Object); !ok {
		if _, isString := obj.(String); !isString || !iterateStrings {
			panic(r.NewTypeError("%s is not an object", obj.String()))
		}
	}
	var iter = obj
	if method := toMethod(r.getV(obj, SymIterator)); method != nil {
		iter = method(FunctionCall{This: obj})
	}
	return r.getIteratorDirect(r.toObject(iter))
}

// stepValue implements IteratorStepValue. It returns false if the iterator is exhausted.
func (ir *iteratorRecord) stepValue() (Value, bool) {
	r := ir.iterator.runtime
	if ir.next == nil {
		panic(r.NewTypeError("iterator.next is missing or not a function"))
	}
	if ir.nextRes != nil {
		return ir.nextRes(_undefined)
	}
	res := r.toObject(ir.next(FunctionCall{This: ir.iterator}))
	if iteratorComplete(res) {
		return nil, false
	}
	return iteratorValue(res), true
}

// closeOnThrow runs f and, if it throws, closes the iterator (ignoring any error from return()) before rethrowing.
func (ir *iteratorRecord) closeOnThrow(f func()) {
	if ex := tryFunc(f); ex != nil {
		_ = tryFunc(ir.returnIter)
		panic(ex)
	}
}

func (ir *iteratorRecord) callOrClose(f func(FunctionCall) Value, args ...Value) (ret Value) {
	ir.closeOnThrow(func() {
		ret = f(FunctionCall{This: _undefined, Arguments: args})
	})
	return
}

func (r *Runtime) thisIteratorObject(call FunctionCall, method string) *Object {
	if obj, ok := call.This.(*Object); ok {
		return obj
	}
	panic(r.NewTypeError("Method Iterator.prototype.%s called on incompatible receiver %s", method, call.This.String()))
}

// iteratorWithCallback implements the common prologue of the helpers taking a callback: the receiver is closed
// if the argument is not callable.
func (r *Runtime) iteratorWithCallback(call FunctionCall, method string) (*iteratorRecord, func(FunctionCall) Value) {
	thisObj := r.thisIteratorObject(call, method)
	arg := call.Argument(0)
	if obj, ok := arg.(*Object); ok {
		if callback, ok := obj.self.assertCallable(); ok {
			return r.getIteratorDirect(thisObj), callback
		}
	}
	(&iteratorRecord{iterator: thisObj}).closeOnThrow(func() {
		panic(r.NewTypeError("%s is not a function", arg.String()))
	})
	return nil, nil
}

// iteratorLimit validates the argument of take() and drop(), closing the receiver on failure.
func (r *Runtime) iteratorLimit(thisObj *Object, arg Value) float64 {
	iterated := &iteratorRecord{iterator: thisObj}
	var limit float64
	iterated.closeOnThrow(func() {
		limit = arg.ToNumber().ToFloat()
		if math.IsNaN(limit) {
			panic(r.newError(r.getRangeError(), "Iterator limit must be a number"))
		}
		limit = math.Trunc(limit)
		if limit < 0 {
			panic(r.newError(r.getRangeError(), "Iterator limit must not be negative"))
		}
	})
	return limit
}

func (r *Runtime) iteratorProto_map(call FunctionCall) Value {
	iterated, mapper := r.iteratorWithCallback(call, "map")
	var counter int64
	return r.newIteratorHelper(iterated, func() (Value, bool) {
		value, ok := iterated.stepValue()
		if !ok {
			return nil, false
		}
		mapped := iterated.callOrClose(mapper, value, intToValue(counter))
		counter++
		return mapped, true
	}).val
}

func (r *Runtime) iteratorProto_filter(call FunctionCall) Value {
	iterated, predicate := r.iteratorWithCallback(call, "filter")
	var counter int64
	return r.newIteratorHelper(iterated, func() (Value, bool) {
		for {
			value, ok := iterated.stepValue()
			if !ok {
				return nil, false
			}
			selected := iterated.callOrClose(predicate, value, intToValue(counter))
			counter++
			if selected.ToBoolean() {
				return value, true
			}
		}
	}).val
}

func (r *Runtime) iteratorProto_take(call FunctionCall) Value {
	thisObj := r.thisIteratorObject(call, "take")
	remaining := r.iteratorLimit(thisObj, call.Argument(0))
	iterated := r.getIteratorDirect(thisObj)
	return r.newIteratorHelper(iterated, func() (Value, bool) {
		if remaining == 0 {
			iterated.returnIter()
			return nil, false
		}
		if !math.IsInf(remaining, 1) {
			remaining--
		}
		return iterated.stepValue()
	}).val
}

func (r *Runtime) iteratorProto_drop(call FunctionCall) Value {
	thisObj := r.thisIteratorObject(call, "drop")
	remaining := r.iteratorLimit(thisObj, call.Argument(0))
	iterated := r.getIteratorDirect(thisObj)
	return r.newIteratorHelper(iterated, func() (Value, bool) {
		for remaining > 0 {
			if !math.IsInf(remaining, 1) {
				remaining--
			}
			if _, valid := iterated.stepValue(); !valid {
				return nil, false
			}
		}
		return iterated.stepValue()
	}).val
}

func (r *Runtime) iteratorProto_flatMap(call FunctionCall) Value {
	iterated, mapper := r.iteratorWithCallback(call, "flatMap")
	var counter int64
	var inner *iteratorRecord
	h := r.newIteratorHelper(iterated, func() (Value, bool) {
		for {
			if inner == nil {
				value, ok := iterated.stepValue()
				if !ok {
					return nil, false
				}
				mapped := iterated.callOrClose(mapper, value, intToValue(counter))
				counter++
				iterated.closeOnThrow(func() {
					inner = r.getIteratorFlattenable(mapped, false)
				})
			}
			var value Value
			var ok bool
			iterated.closeOnThrow(func() {
				value, ok = inner.stepValue()
			})
			if ok {
				return value, true
			}
			inner = nil
		}
	})
	h.abort = func() {
		if inner != nil {
			iterated.closeOnThrow(inner.returnIter)
		}
		iterated.returnIter()
	}
	return h.val
}

func (r *Runtime) iteratorProto_reduce(call FunctionCall) Value {
	iterated, reducer := r.iteratorWithCallback(call, "reduce")
	var accumulator Value
	var counter int64
	if len(call.Arguments) > 1 {
		accumulator = call.Arguments[1]
	} else {
		var ok bool
		accumulator, ok = iterated.stepValue()
		if !ok {
			panic(r.NewTypeError("Reduce of empty iterator with no initial value"))
		}
		counter = 1
	}
	for {
		value, ok := iterated.stepValue()
		if !ok {
			return accumulator
		}
		accumulator = iterated.callOrClose(reducer, accumulator, value, intToValue(counter))
		counter++
	}
}

func (r *Runtime) iteratorProto_toArray(call FunctionCall) Value {
	iterated := r.getIteratorDirect(r.thisIteratorObject(call, "toArray"))
	var items []Value
	for {
		value, ok := iterated.stepValue()
		if !ok {
			return r.newArrayValues(items)
		}
		items = append(items, value)
	}
}

func (r *Runtime) iteratorProto_forEach(call FunctionCall) Value {
	iterated, fn := r.iteratorWithCallback(call, "forEach")
	for counter := int64(0); ; counter++ {
		value, ok := iterated.stepValue()
		if !ok {
			return _undefined
		}
		iterated.callOrClose(fn, value, intToValue(counter))
	}
}

func (r *Runtime) iteratorProto_some(call FunctionCall) Value {
	iterated, predicate := r.iteratorWithCallback(call, "some")
	for counter := int64(0); ; counter++ {
		value, ok := iterated.stepValue()
		if !ok {
			return valueFalse
		}
		if iterated.callOrClose(predicate, value, intToValue(counter)).ToBoolean() {
			iterated.returnIter()
			return valueTrue
		}
	}
}

func (r *Runtime) iteratorProto_every(call FunctionCall) Value {
	iterated, predicate := r.iteratorWithCallback(call, "every")
	for counter := int64(0); ; counter++ {
		value, ok := iterated.stepValue()
		if !ok {
			return valueTrue
		}
		if !iterated.callOrClose(predicate, value, intToValue(counter)).ToBoolean() {
			iterated.returnIter()
			return valueFalse
		}
	}
}

func (r *Runtime) iteratorProto_find(call FunctionCall) Value {
	iterated, predicate := r.iteratorWithCallback(call, "find")
	for counter := int64(0); ; counter++ {
		value, ok := iterated.stepValue()
		if !ok {
			return _undefined
		}
		if iterated.callOrClose(predicate, value, intToValue(counter)).ToBoolean() {
			iterated.returnIter()
			return value
		}
	}
}

func (r *Runtime) iteratorHelperProto_next(call FunctionCall) Value {
	if obj, ok := call.This.(*Object); ok {
		if h, ok := obj.self.(*iteratorHelperObject); ok {
			return h.next()
		}
	}
	panic(r.NewTypeError("Method Iterator Helper.prototype.next called on incompatible receiver"))
}

func (r *Runtime) iteratorHelperProto_return(call FunctionCall) Value {
	if obj, ok := call.This.(*Object); ok {
		if h, ok := obj.self.(*iteratorHelperObject); ok {
			return h._return()
		}
	}
	panic(r.NewTypeError("Method Iterator Helper.prototype.return called on incompatible receiver"))
}

func (r *Runtime) wrapForValidIteratorProto_next(call FunctionCall) Value {
	if obj, ok := call.This.(*Object); ok {
		if w, ok := obj.self.(*wrapForValidIteratorObject); ok {
			if w.iterated.next == nil {
				panic(r.NewTypeError("iterator.next is missing or not a function"))
			}
			return w.iterated.next(FunctionCall{This: w.iterated.iterator})
		}
	}
	panic(r.NewTypeError("Method %%WrapForValidIteratorPrototype%%.next called on incompatible receiver"))
}

func (r *Runtime) wrapForValidIteratorProto_return(call FunctionCall) Value {
	if obj, ok := call.This.(*Object); ok {
		if w, ok := obj.self.(*wrapForValidIteratorObject); ok {
			iter := w.iterated.iterator
			if method := toMethod(iter.self.getStr("return", nil)); method != nil {
				return method(FunctionCall{This: iter})
			}
			return r.createIterResultObject(_undefined, true)
		}
	}
	panic(r.NewTypeError("Method %%WrapForValidIteratorPrototype%%.return called on incompatible receiver"))
}

func (r *Runtime) iterator_from(call FunctionCall) Value {
	iterated := r.getIteratorFlattenable(call.Argument(0), true)
	if hasInstance(r.getIteratorCtor(), iterated.iterator) {
		return iterated.iterator
	}
	w := &wrapForValidIteratorObject{iterated: iterated}
	o := &Object{runtime: r}
	w.class = classObject
	w.val = o
	w.extensible = true
	o.self = w
	w.prototype = r.getWrapForValidIteratorPrototype()
	w.init()
	return o
}

func (r *Runtime) builtin_newIterator(_ []Value, newTarget *Object) *Object {
	if newTarget == nil || newTarget == r.global.Iterator {
		panic(r.NewTypeError("Abstract class Iterator not directly constructable"))
	}
	proto := r.getPrototypeFromCtor(newTarget, r.global.Iterator, r.getIteratorPrototype())
	return r.newBaseObject(proto, classObject).val
}

// setterThatIgnoresPrototypeProperties implements SetterThatIgnoresPrototypeProperties. It is used by the
// accessors on Iterator.prototype, which are accessors only for web compatibility.
func (r *Runtime) setterThatIgnoresPrototypeProperties(this Value, home *Object, homeName string, p Value, v Value) {
	obj, ok := this.(*Object)
	if !ok {
		panic(r.NewTypeError("%s setter called on incompatible receiver %s", homeName, this.String()))
	}
	if obj == home {
		panic(r.NewTypeError("Cannot assign to a read-only property of %s", homeName))
	}
	if obj.getOwnProp(p) == nil {
		createDataPropertyOrThrow(obj, p, v)
	} else {
		obj.set(p, v, obj, true)
	}
}

func (r *Runtime) iteratorProtoAccessor(key Value, get func() Value, name string) Value {
	return &valueProperty{
		accessor: true,
		getterFunc: r.newNativeFunc(func(FunctionCall) Value {
			return get()
		}, unistring.String("get "+name), 0),
		setterFunc: r.newNativeFunc(func(call FunctionCall) Value {
			r.setterThatIgnoresPrototypeProperties(call.This, r.getIteratorPrototype(), "Iterator.prototype", key, call.Argument(0))
			return _undefined
		}, unistring.String("set "+name), 1),
		configurable: true,
	}
}

func (r *Runtime) addIteratorHelpers(o *baseObject) {
	o._putProp("map", r.newNativeFunc(r.iteratorProto_map, "map", 1), true, false, true)
	o._putProp("filter", r.newNativeFunc(r.iteratorProto_filter, "filter", 1), true, false, true)
	o._putProp("take", r.newNativeFunc(r.iteratorProto_take, "take", 1), true, false, true)
	o._putProp("drop", r.newNativeFunc(r.iteratorProto_drop, "drop", 1), true, false, true)
	o._putProp("flatMap", r.newNativeFunc(r.iteratorProto_flatMap, "flatMap", 1), true, false, true)
	o._putProp("reduce", r.newNativeFunc(r.iteratorProto_reduce, "reduce", 1), true, false, true)
	o._putProp("toArray", r.newNativeFunc(r.iteratorProto_toArray, "toArray", 0), true, false, true)
	o._putProp("forEach", r.newNativeFunc(r.iteratorProto_forEach, "forEach", 1), true, false, true)
	o._putProp("some", r.newNativeFunc(r.iteratorProto_some, "some", 1), true, false, true)
	o._putProp("every", r.newNativeFunc(r.iteratorProto_every, "every", 1), true, false, true)
	o._putProp("find", r.newNativeFunc(r.iteratorProto_find, "find", 1), true, false, true)

	o._put("constructor", r.iteratorProtoAccessor(asciiString("constructor"), func() Value {
		return r.getIteratorCtor()
	}, "constructor"))
	o._putSym(SymToStringTag, r.iteratorProtoAccessor(SymToStringTag, func() Value {
		return asciiString(classIterator)
	}, "[Symbol.toStringTag]"))
}

func (r *Runtime) createIteratorHelperProto(val *Object) objectImpl {
	o := newBaseObjectObj(val, r.getIteratorPrototype(), classObject)

	o._putProp("next", r.newIteratorNextFunc(r.iteratorHelperProto_next, 0, func(iterator *Object) func(Value) (Value, bool) {
		if i, ok := iterator.self.(*iteratorHelperObject); ok {
			return i.nextResult
		}
		return nil
	}), true, false, true)
	o._putProp("return", r.newNativeFunc(r.iteratorHelperProto_return, "return", 0), true, false, true)
	o._putSym(SymToStringTag, valueProp(asciiString(classIteratorHelper), false, false, true))

	return o
}

func (r *Runtime) getIteratorHelperPrototype() *Object {
	var o *Object
	if o = r.global.IteratorHelperPrototype; o == nil {
		o = &Object{runtime: r}
		r.global.IteratorHelperPrototype = o
		o.self = r.createIteratorHelperProto(o)
	}
	return o
}

func (r *Runtime) createWrapForValidIteratorProto(val *Object) objectImpl {
	o := newBaseObjectObj(val, r.getIteratorPrototype(), classObject)

	o._putProp("next", r.newIteratorNextFunc(r.wrapForValidIteratorProto_next, 0, func(iterator *Object) func(Value) (Value, bool) {
		if i, ok := iterator.self.(*wrapForValidIteratorObject); ok {
			return i.iterated.nextRes
		}
		return nil
	}), true, false, true)
	o._putProp("return", r.newNativeFunc(r.wrapForValidIteratorProto_return, "return", 0), true, false, true)

	return o
}

func (r *Runtime) getWrapForValidIteratorPrototype() *Object {
	var o *Object
	if o = r.global.WrapForValidIteratorPrototype; o == nil {
		o = &Object{runtime: r}
		r.global.WrapForValidIteratorPrototype = o
		o.self = r.createWrapForValidIteratorProto(o)
	}
	return o
}

func (r *Runtime) createIteratorCtor(val *Object) objectImpl {
	o := r.newNativeConstructOnly(val, r.builtin_newIterator, r.getIteratorPrototype(), "Iterator", 0)
	o._putProp("from", r.newNativeFunc(r.iterator_from, "from", 1), true, false, true)

	return o
}

func (r *Runtime) getIteratorCtor() *Object {
	ret := r.global.Iterator
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.Iterator = ret
		ret.self = r.createIteratorCtor(ret)
	}
	return ret
}
