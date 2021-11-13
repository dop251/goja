package goja

func (r *Runtime) bigintproto_valueOf(call FunctionCall) Value {
	this := call.This
	if !isBigInt(this) {
		r.typeErrorResult(true, "Value is not a bigint")
	}
	switch t := this.(type) {
	case valueBigInt:
		return this
	case *Object:
		if v, ok := t.self.(*primitiveValueObject); ok {
			return v.pValue
		}
	}

	panic(r.NewTypeError("BigInt.prototype.valueOf is not generic"))
}

func isBigInt(v Value) bool {
	switch t := v.(type) {
	case valueBigInt:
		return true
	case *Object:
		switch t := t.self.(type) {
		case *primitiveValueObject:
			return isBigInt(t.pValue)
		}
	}
	return false
}

func (r *Runtime) bigintproto_toString(call FunctionCall) Value {
	this := call.This
	if !isBigInt(this) {
		r.typeErrorResult(true, "Value is not a bigint")
	}
	b := call.This.ToBigInt()
	if t, ok := b.(valueBigInt); ok {
		return asciiString(t.Int.String())
	}
	panic(r.NewTypeError("BigInt.prototype.toString is not generic"))
}

func (r *Runtime) initBigInt() {
	r.global.BigIntPrototype = r.newPrimitiveObject(valueInt(0), r.global.ObjectPrototype, classBigInt)
	o := r.global.BigIntPrototype.self
	o._putProp("toString", r.newNativeFunc(r.bigintproto_toString, nil, "toString", nil, 1), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.bigintproto_valueOf, nil, "valueOf", nil, 0), true, false, true)

	r.global.BigInt = r.newNativeFunc(r.builtin_BigInt, r.builtin_newBigInt, "BigInt", r.global.BigIntPrototype, 1)
	r.addToGlobal("BigInt", r.global.BigInt)
}
