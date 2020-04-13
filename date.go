package goja

import (
	"time"
)

const (
	dateTimeLayout       = "Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"
	utcDateTimeLayout    = "Mon, 02 Jan 2006 15:04:05 GMT"
	isoDateTimeLayout    = "2006-01-02T15:04:05.000Z"
	dateLayout           = "Mon Jan 02 2006"
	timeLayout           = "15:04:05 GMT-0700 (MST)"
	datetimeLayout_en_GB = "01/02/2006, 15:04:05"
	dateLayout_en_GB     = "01/02/2006"
	timeLayout_en_GB     = "15:04:05"
)

type dateObject struct {
	baseObject
	time  time.Time
	isSet bool
}

var (
	dateLayoutList = []string{
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC1123,
		time.RFC1123Z,
		dateTimeLayout,
		time.UnixDate,
		time.ANSIC,
		time.RubyDate,
		"Mon, 02 Jan 2006 15:04:05 GMT-0700 (MST)",
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",

		"2006",
		"2006-01",

		"2006T15:04",
		"2006-01T15:04",
		"2006-01-02T15:04",

		"2006T15:04:05",
		"2006-01T15:04:05",

		"2006T15:04Z0700",
		"2006-01T15:04Z0700",
		"2006-01-02T15:04Z0700",

		"2006T15:04:05Z0700",
		"2006-01T15:04:05Z0700",
	}
)

func dateParse(date string) (time.Time, bool) {
	var t time.Time
	var err error
	for _, layout := range dateLayoutList {
		t, err = parseDate(layout, date, time.UTC)
		if err == nil {
			break
		}
	}
	unix := timeToMsec(t)
	return t, err == nil && unix >= -8640000000000000 && unix <= 8640000000000000
}

func (r *Runtime) newDateObject(t time.Time, isSet bool, proto *Object) *Object {
	v := &Object{runtime: r}
	d := &dateObject{}
	v.self = d
	d.val = v
	d.class = classDate
	d.prototype = proto
	d.extensible = true
	d.init()
	d.time = t.In(time.Local)
	d.isSet = isSet
	return v
}

func dateFormat(t time.Time) string {
	return t.Local().Format(dateTimeLayout)
}

func (d *dateObject) toPrimitive() Value {
	return d.toPrimitiveString()
}

func (d *dateObject) export() interface{} {
	if d.isSet {
		return d.time
	}
	return nil
}

func (d *dateObject) setTime(year, m, day, hour, min, sec, nsec int64) Value {
	t, ok := mkTime(year, m, day, hour, min, sec, nsec, time.Local)
	if ok {
		return d.setTimeMs(timeToMsec(t))
	}
	d.isSet = false
	return _NaN
}

func (d *dateObject) setTimeUTC(year, m, day, hour, min, sec, nsec int64) Value {
	t, ok := mkTime(year, m, day, hour, min, sec, nsec, time.UTC)
	if ok {
		t = t.In(time.Local)
		return d.setTimeMs(timeToMsec(t))
	}
	d.isSet = false
	return _NaN
}

func (d *dateObject) setTimeMs(ms int64) Value {
	if ms >= 0 && ms <= maxTime || ms < 0 && ms >= -maxTime {
		d.time = timeFromMsec(ms)
		return intToValue(ms)
	}
	d.isSet = false
	return _NaN
}
