package goja

type argumentsObject struct {
	baseObject
	length int
}

type mappedProperty struct {
	valueProperty
	v *Value
}

func (a *argumentsObject) get(p Value, receiver Value) Value {
	if s, ok := p.(*valueSymbol); ok {
		return a.getSym(s, receiver)
	}
	return a.getStr(p.String(), receiver)
}

func (a *argumentsObject) getStr(name string, receiver Value) Value {
	return a.getFromProp(a.getPropStr(name), receiver)
}

func (a *argumentsObject) getProp(n Value) Value {
	if s, ok := n.(*valueSymbol); ok {
		return a.getPropSym(s)
	}
	return a.getPropStr(n.String())
}

func (a *argumentsObject) getPropStr(name string) Value {
	if val := a.getOwnPropStr(name); val != nil {
		return val
	}
	return a.getProtoPropStr(name)
}

func (a *argumentsObject) getOwnPropStr(name string) Value {
	if mapped, ok := a.values[name].(*mappedProperty); ok {
		return *mapped.v
	}

	return a.baseObject.getOwnPropStr(name)
}

func (a *argumentsObject) getOwnProp(name Value) Value {
	if s, ok := name.(*valueSymbol); ok {
		return a.getOwnPropSym(s)
	}

	return a.getOwnPropStr(name.String())
}

func (a *argumentsObject) init() {
	a.baseObject.init()
	a._putProp("length", intToValue(int64(a.length)), true, false, true)
}

func (a *argumentsObject) put(n Value, val Value, throw bool) {
	if s, ok := n.(*valueSymbol); ok {
		a.putSym(s, val, throw)
		return
	}
	a.putStr(n.String(), val, throw)
}

func (a *argumentsObject) putStr(name string, val Value, throw bool) {
	if prop, ok := a.values[name].(*mappedProperty); ok {
		if !prop.writable {
			a.val.runtime.typeErrorResult(throw, "Property is not writable: %s", name)
			return
		}
		*prop.v = val
		return
	}
	a.baseObject.putStr(name, val, throw)
}

func (a *argumentsObject) deleteStr(name string, throw bool) bool {
	if prop, ok := a.values[name].(*mappedProperty); ok {
		if !a.checkDeleteProp(name, &prop.valueProperty, throw) {
			return false
		}
		a._delete(name)
		return true
	}

	return a.baseObject.deleteStr(name, throw)
}

func (a *argumentsObject) delete(n Value, throw bool) bool {
	if s, ok := n.(*valueSymbol); ok {
		return a.deleteSym(s, throw)
	}
	return a.deleteStr(n.String(), throw)
}

type argumentsPropIter struct {
	wrapped iterNextFunc
}

func (i *argumentsPropIter) next() (propIterItem, iterNextFunc) {
	var item propIterItem
	item, i.wrapped = i.wrapped()
	if i.wrapped == nil {
		return propIterItem{}, nil
	}
	if prop, ok := item.value.(*mappedProperty); ok {
		item.value = *prop.v
	}
	return item, i.next
}

func (a *argumentsObject) _enumerate(recursive bool) iterNextFunc {
	return (&argumentsPropIter{
		wrapped: a.baseObject._enumerate(recursive),
	}).next

}

func (a *argumentsObject) enumerate(all, recursive bool) iterNextFunc {
	return (&argumentsPropIter{
		wrapped: a.baseObject.enumerate(all, recursive),
	}).next
}

func (a *argumentsObject) defineOwnProperty(n Value, descr PropertyDescriptor, throw bool) bool {
	if _, ok := n.(*valueSymbol); ok {
		return a.baseObject.defineOwnProperty(n, descr, throw)
	}
	name := n.String()
	if mapped, ok := a.values[name].(*mappedProperty); ok {
		existing := &valueProperty{
			configurable: mapped.configurable,
			writable:     true,
			enumerable:   mapped.enumerable,
			value:        mapped.get(a.val),
		}

		val, ok := a.baseObject._defineOwnProperty(n, existing, descr, throw)
		if !ok {
			return false
		}

		if prop, ok := val.(*valueProperty); ok {
			if !prop.accessor {
				*mapped.v = prop.value
			}
			if prop.accessor || !prop.writable {
				a._put(name, prop)
				return true
			}
			mapped.configurable = prop.configurable
			mapped.enumerable = prop.enumerable
		} else {
			*mapped.v = val
			mapped.configurable = true
			mapped.enumerable = true
		}

		return true
	}

	return a.baseObject.defineOwnProperty(n, descr, throw)
}

func (a *argumentsObject) export() interface{} {
	arr := make([]interface{}, a.length)
	for i := range arr {
		v := a.get(intToValue(int64(i)), nil)
		if v != nil {
			arr[i] = v.Export()
		}
	}
	return arr
}
