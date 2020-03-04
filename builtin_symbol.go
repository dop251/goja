package goja

var (
	symHasInstance        = &valueSymbol{desc: "Symbol.hasInstance"}
	symIsConcatSpreadable = &valueSymbol{desc: "Symbol.isConcatSpreadable"}
	symIterator           = &valueSymbol{desc: "Symbol.iterator"}
	symMatch              = &valueSymbol{desc: "Symbol.match"}
	symReplace            = &valueSymbol{desc: "Symbol.replace"}
	symSearch             = &valueSymbol{desc: "Symbol.search"}
	symSpecies            = &valueSymbol{desc: "Symbol.species"}
	symSplit              = &valueSymbol{desc: "Symbol.split"}
	symToPrimitive        = &valueSymbol{desc: "Symbol.toPrimitive"}
	symToStringTag        = &valueSymbol{desc: "Symbol.toStringTag"}
	symUnscopables        = &valueSymbol{desc: "Symbol.unscopables"}
)

func (r *Runtime) builtin_symbol(call FunctionCall) Value {
	desc := ""
	if arg := call.Argument(0); !IsUndefined(arg) {
		desc = arg.toString().String()
	}
	return &valueSymbol{
		desc: desc,
	}
}

func (r *Runtime) symbolproto_tostring(call FunctionCall) Value {
	sym, ok := call.This.(*valueSymbol)
	if !ok {
		if obj, ok := call.This.(*Object); ok {
			if v, ok := obj.self.(*primitiveValueObject); ok {
				if sym1, ok := v.pValue.(*valueSymbol); ok {
					sym = sym1
				}
			}
		}
	}
	if sym == nil {
		panic(r.NewTypeError("Method Symbol.prototype.toString is called on incompatible receiver"))
	}
	return newStringValue(sym.descString())
}

func (r *Runtime) symbolproto_valueOf(call FunctionCall) Value {
	_, ok := call.This.(*valueSymbol)
	if ok {
		return call.This
	}

	if obj, ok := call.This.(*Object); ok {
		if v, ok := obj.self.(*primitiveValueObject); ok {
			if sym, ok := v.pValue.(*valueSymbol); ok {
				return sym
			}
		}
	}

	panic(r.NewTypeError("Symbol.prototype.valueOf requires that 'this' be a Symbol"))
}

func (r *Runtime) symbol_for(call FunctionCall) Value {
	key := call.Argument(0).toString().String()
	if v := r.symbolRegistry[key]; v != nil {
		return v
	}
	if r.symbolRegistry == nil {
		r.symbolRegistry = make(map[string]*valueSymbol)
	}
	v := &valueSymbol{
		desc: key,
	}
	r.symbolRegistry[key] = v
	return v
}

func (r *Runtime) symbol_keyfor(call FunctionCall) Value {
	arg := call.Argument(0)
	sym, ok := arg.(*valueSymbol)
	if !ok {
		panic(r.NewTypeError("%s is not a symbol", arg.String()))
	}
	for key, s := range r.symbolRegistry {
		if s == sym {
			return r.ToValue(key)
		}
	}
	return _undefined
}

func (r *Runtime) createSymbolProto(val *Object) objectImpl {
	o := &baseObject{
		class:      classObject,
		val:        val,
		extensible: true,
		prototype:  r.global.ObjectPrototype,
	}
	o.init()

	o._putProp("constructor", r.global.Symbol, true, false, true)
	o._putProp("toString", r.newNativeFunc(r.symbolproto_tostring, nil, "toString", nil, 0), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.symbolproto_valueOf, nil, "valueOf", nil, 0), true, false, true)
	o.putSym(symToPrimitive, valueProp(r.newNativeFunc(r.symbolproto_valueOf, nil, "[Symbol.toPrimitive]", nil, 1), false, false, true), true)
	o.putSym(symToStringTag, valueProp(newStringValue("Symbol"), false, false, true), true)

	return o
}

func (r *Runtime) createSymbol(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.builtin_symbol, nil, "Symbol", r.global.SymbolPrototype, 0)

	o._putProp("for", r.newNativeFunc(r.symbol_for, nil, "for", nil, 1), true, false, true)
	o._putProp("keyFor", r.newNativeFunc(r.symbol_keyfor, nil, "keyFor", nil, 1), true, false, true)

	for _, s := range []*valueSymbol{
		symHasInstance,
		symIsConcatSpreadable,
		symIterator,
		symMatch,
		symReplace,
		symSearch,
		symSpecies,
		symSplit,
		symToPrimitive,
		symToStringTag,
		symUnscopables,
	} {
		o._putProp(s.desc[len("Symbol."):], s, false, false, false)
	}

	return o
}

func (r *Runtime) initSymbol() {
	r.global.SymbolPrototype = r.newLazyObject(r.createSymbolProto)

	r.global.Symbol = r.newLazyObject(r.createSymbol)
	r.addToGlobal("Symbol", r.global.Symbol)

}
