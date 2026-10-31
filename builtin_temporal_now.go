package goja

import (
	"math/big"
	"time"
)

// getTemporalNow builds the Temporal.Now namespace.
func (r *Runtime) getTemporalNow() *Object {
	o := r.NewObject()
	o.Set("instant", r.newNativeFunc(r.temporalNowInstant, "instant", 0))
	o.Set("timeZoneId", r.newNativeFunc(r.temporalNowTimeZoneID, "timeZoneId", 0))
	o.Set("zonedDateTimeISO", r.newNativeFunc(r.temporalNowZonedDateTimeISO, "zonedDateTimeISO", 0))
	o.Set("plainDateTimeISO", r.newNativeFunc(r.temporalNowPlainDateTimeISO, "plainDateTimeISO", 0))
	o.Set("plainDateISO", r.newNativeFunc(r.temporalNowPlainDateISO, "plainDateISO", 0))
	o.Set("plainTimeISO", r.newNativeFunc(r.temporalNowPlainTimeISO, "plainTimeISO", 0))
	return o
}

func (r *Runtime) temporalNowEpochNs() *big.Int {
	now := time.Now()
	ns := new(big.Int).Mul(big.NewInt(now.Unix()), big.NewInt(1e9))
	return ns.Add(ns, big.NewInt(int64(now.Nanosecond())))
}

// temporalNowZone returns the requested zone or the system time zone.
func (r *Runtime) temporalNowZone(call FunctionCall) *temporalZone {
	v := call.Argument(0)
	if v == nil || IsUndefined(v) {
		return &temporalZone{id: time.Local.String(), loc: time.Local}
	}
	return r.parseTemporalZone(v)
}

func (r *Runtime) temporalNowInstant(call FunctionCall) Value {
	return r.newTemporalInstantObject(r.temporalNowEpochNs(), r.getTemporalInstantPrototype())
}

func (r *Runtime) temporalNowTimeZoneID(call FunctionCall) Value {
	zone := r.temporalNowZone(call)
	return r.ToValue(zone.id)
}

func (r *Runtime) temporalNowZonedDateTimeISO(call FunctionCall) Value {
	zone := r.temporalNowZone(call)
	return r.newTemporalZonedDateTimeObject(r.temporalNowEpochNs(), zone, "iso8601", r.getTemporalZonedDateTimePrototype())
}

func (r *Runtime) temporalNowPlainDateTimeISO(call FunctionCall) Value {
	zone := r.temporalNowZone(call)
	d, t, _ := zone.localDateTime(r.temporalNowEpochNs())
	return r.newTemporalPlainDateTimeObject(d, t, "iso8601", r.getTemporalPlainDateTimePrototype())
}

func (r *Runtime) temporalNowPlainDateISO(call FunctionCall) Value {
	zone := r.temporalNowZone(call)
	d, _, _ := zone.localDateTime(r.temporalNowEpochNs())
	return r.newTemporalPlainDateObject(d, "iso8601", r.getTemporalPlainDatePrototype())
}

func (r *Runtime) temporalNowPlainTimeISO(call FunctionCall) Value {
	zone := r.temporalNowZone(call)
	_, t, _ := zone.localDateTime(r.temporalNowEpochNs())
	return r.newTemporalPlainTimeObject(t.hour, t.minute, t.second, t.millisecond, t.microsecond, t.nanosecond, r.getTemporalPlainTimePrototype())
}
