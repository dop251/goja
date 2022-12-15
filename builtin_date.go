package goja

import (
	"math"
	"time"
)

// _norm returns nhi, nlo such that
//
//	hi * base + lo == nhi * base + nlo
//	0 <= nlo < base
func _norm(hi, lo, base int64) (nhi, nlo int64, ok bool) {
	if lo < 0 {
		if hi == math.MinInt64 && lo <= -base {
			// underflow
			ok = false
			return
		}
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		if hi == math.MaxInt64 {
			// overflow
			ok = false
			return
		}
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo, true
}

func mkTime(year, m, day, hour, min, sec, nsec int64, loc *time.Location) (t time.Time, ok bool) {
	year, m, ok = _norm(year, m, 12)
	if !ok {
		return
	}

	// Normalise nsec, sec, min, hour, overflowing into day.
	sec, nsec, ok = _norm(sec, nsec, 1e9)
	if !ok {
		return
	}
	min, sec, ok = _norm(min, sec, 60)
	if !ok {
		return
	}
	hour, min, ok = _norm(hour, min, 60)
	if !ok {
		return
	}
	day, hour, ok = _norm(day, hour, 24)
	if !ok {
		return
	}
	if year > math.MaxInt32 || year < math.MinInt32 ||
		day > math.MaxInt32 || day < math.MinInt32 ||
		m >= math.MaxInt32 || m < math.MinInt32-1 {
		return time.Time{}, false
	}
	month := time.Month(m) + 1
	return time.Date(int(year), month, int(day), int(hour), int(min), int(sec), int(nsec), loc), true
}

func _intArg(call FunctionCall, argNum int) (int64, bool) {
	n := call.Argument(argNum).ToNumber()
	if IsNaN(n) {
		return 0, false
	}
	return n.ToInteger(), true
}

func _dateSetYear(t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var year int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		year, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
		if year >= 0 && year <= 99 {
			year += 1900
		}
	} else {
		year = int64(t.Year())
	}

	return _dateSetMonth(year, t, call, argNum+1, utc)
}

func _dateSetFullYear(t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var year int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		year, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		year = int64(t.Year())
	}
	return _dateSetMonth(year, t, call, argNum+1, utc)
}

func _dateSetMonth(year int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var mon int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		mon, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		mon = int64(t.Month()) - 1
	}

	return _dateSetDay(year, mon, t, call, argNum+1, utc)
}

func _dateSetDay(year, mon int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var day int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		day, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		day = int64(t.Day())
	}

	return _dateSetHours(year, mon, day, t, call, argNum+1, utc)
}

func _dateSetHours(year, mon, day int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var hours int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		hours, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		hours = int64(t.Hour())
	}
	return _dateSetMinutes(year, mon, day, hours, t, call, argNum+1, utc)
}

func _dateSetMinutes(year, mon, day, hours int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var min int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		min, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		min = int64(t.Minute())
	}
	return _dateSetSeconds(year, mon, day, hours, min, t, call, argNum+1, utc)
}

func _dateSetSeconds(year, mon, day, hours, min int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var sec int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		sec, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		sec = int64(t.Second())
	}
	return _dateSetMilliseconds(year, mon, day, hours, min, sec, t, call, argNum+1, utc)
}

func _dateSetMilliseconds(year, mon, day, hours, min, sec int64, t time.Time, call FunctionCall, argNum int, utc bool) (time.Time, bool) {
	var msec int64
	if argNum == 0 || argNum > 0 && argNum < len(call.Arguments) {
		var ok bool
		msec, ok = _intArg(call, argNum)
		if !ok {
			return time.Time{}, false
		}
	} else {
		msec = int64(t.Nanosecond() / 1e6)
	}
	var ok bool
	sec, msec, ok = _norm(sec, msec, 1e3)
	if !ok {
		return time.Time{}, false
	}

	var loc *time.Location
	if utc {
		loc = time.UTC
	} else {
		loc = time.Local
	}
	r, ok := mkTime(year, mon, day, hours, min, sec, msec*1e6, loc)
	if !ok {
		return time.Time{}, false
	}
	if utc {
		return r.In(time.Local), true
	}
	return r, true
}

func (r *Runtime) dateproto_setMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		n := call.Argument(0).ToNumber()
		if IsNaN(n) {
			d.unset()
			return Null()
		}
		msec := n.ToInteger()
		sec := d.msec / 1e3
		var ok bool
		sec, msec, ok = _norm(sec, msec, 1e3)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(sec*1e3 + msec)
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setMilliseconds is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		n := call.Argument(0).ToNumber()
		if IsNaN(n) {
			d.unset()
			return Null()
		}
		msec := n.ToInteger()
		sec := d.msec / 1e3
		var ok bool
		sec, msec, ok = _norm(sec, msec, 1e3)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(sec*1e3 + msec)
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCMilliseconds is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.time(), call, -5, false)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setSeconds is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.timeUTC(), call, -5, true)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCSeconds is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.time(), call, -4, false)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setMinutes is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.timeUTC(), call, -4, true)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCMinutes is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.time(), call, -3, false)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setHours is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.timeUTC(), call, -3, true)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCHours is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.time(), limitCallArgs(call, 1), -2, false)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setDate is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.timeUTC(), limitCallArgs(call, 1), -2, true)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCDate is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.time(), limitCallArgs(call, 2), -1, false)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setMonth is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setUTCMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		t, ok := _dateSetFullYear(d.timeUTC(), limitCallArgs(call, 2), -1, true)
		if !ok {
			d.unset()
			return Null()
		}
		if d.isSet() {
			return d.setTimeMs(timeToMsec(t))
		} else {
			return Null()
		}
	}
	panic(r.NewTypeError("Method Date.prototype.setUTCMonth is called on incompatible receiver"))
}

func (r *Runtime) dateproto_setFullYear(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		var t time.Time
		if d.isSet() {
			t = d.time()
		} else {
			t = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.Local)
		}
		t, ok := _dateSetFullYear(t, limitCallArgs(call, 3), 0, false)
		if !ok {
			d.unset()
			return Null()
		}
		return d.setTimeMs(timeToMsec(t))
	}
	panic(r.NewTypeError("Method Date.prototype.setFullYear is called on incompatible receiver"))
}
