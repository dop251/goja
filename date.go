package goja

import (
	"math"
	"reflect"
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

	maxTime   = 8.64e15
	timeUnset = math.MinInt64
)

var datetimeLayout_locales_GB = map[string]string{
	"cs-CZ": "2. 1. 2006",
	"da-DK": "2.1.2006",
	"de-AT": "2.1.2006",
	"de-CH": "2.1.2006",
	"de-DE": "2.1.2006",
	"el-GR": "2/1/2006",
	"en-AU": "02/01/2006",
	"en-CA": "2006-01-02",
	"en-GB": "02/01/2006",
	"en-IE": "2/1/2006",
	"en-IN": "2/1/2006",
	"en-NZ": "2/01/2006",
	"en-US": "1/2/2006",
	"en-ZA": "2006/01/02",
	"es-AR": "2/1/2006",
	"es-CL": "02-01-2006",
	"es-CO": "2/1/2006",
	"es-ES": "2/1/2006",
	"es-MX": "2/1/2006",
	"es-US": "2/1/2006",
	"fi-FI": "2.1.2006",
	"fr-BE": "02/01/2006",
	"fr-CA": "2006-01-02",
	"fr-CH": "02.01.2006",
	"fr-FR": "02/01/2006",
	"he-IL": "2.1.2006",
	"hi-IN": "2/1/2006",
	"hu-HU": "2006. 01. 02.",
	"id-ID": "2/1/2006",
	"it-CH": "02/01/2006",
	"it-IT": "02/01/2006",
	"ja-JP": "2006/1/2",
	"ko-KR": "2006. 1. 2.",
	"nl-BE": "2/1/2006",
	"nl-NL": "2-1-2006",
	"no-NO": "2.1.2006",
	"pl-PL": "2.01.2006",
	"pt-BR": "02/01/2006",
	"pt-PT": "02/01/2006",
	"ro-RO": "02.01.2006",
	"ru-RU": "02.01.2006",
	"sk-SK": "2. 1. 2006",
	"sv-SE": "2006-01-02",
	"ta-IN": "2/1/2006",
	"ta-LK": "2/1/2006",
	"th-TH": "2/1/2006",
	"tr-TR": "02.01.2006",
	"zh-CN": "2006/1/2",
	"zh-HK": "2/1/2006",
	"zh-TW": "2006/1/2",
}

type dateObject struct {
	baseObject
	msec int64
}

func dateParse(date string) (t time.Time, ok bool) {
	d, ok := parseDateISOString(date)
	if !ok {
		d, ok = parseDateOtherString(date)
	}
	if !ok {
		return
	}
	if d.month > 12 ||
		d.day > 31 ||
		d.hour > 24 ||
		d.min > 59 ||
		d.sec > 59 ||
		// special case 24:00:00.000
		(d.hour == 24 && (d.min != 0 || d.sec != 0 || d.msec != 0)) {
		ok = false
		return
	}
	var loc *time.Location
	if d.isLocal {
		loc = time.Local
	} else {
		loc = time.FixedZone("", d.timeZoneOffset*60)
	}
	t = time.Date(d.year, time.Month(d.month), d.day, d.hour, d.min, d.sec, d.msec*1e6, loc)
	unixMilli := t.UnixMilli()
	ok = unixMilli >= -maxTime && unixMilli <= maxTime
	return
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
	if isSet {
		d.msec = timeToMsec(t)
	} else {
		d.msec = timeUnset
	}
	return v
}

func dateFormat(t time.Time) string {
	return t.Local().Format(dateTimeLayout)
}

func timeFromMsec(msec int64) time.Time {
	sec := msec / 1000
	nsec := (msec % 1000) * 1e6
	return time.Unix(sec, nsec)
}

func timeToMsec(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond())/1e6
}

func (d *dateObject) exportType() reflect.Type {
	return typeTime
}

func (d *dateObject) export(*objectExportCtx) interface{} {
	if d.isSet() {
		return d.time()
	}
	return nil
}

func (d *dateObject) setTimeMs(ms int64) Value {
	if ms >= 0 && ms <= maxTime || ms < 0 && ms >= -maxTime {
		d.msec = ms
		return intToValue(ms)
	}

	d.unset()
	return _NaN
}

func (d *dateObject) isSet() bool {
	return d.msec != timeUnset
}

func (d *dateObject) unset() {
	d.msec = timeUnset
}

func (d *dateObject) time() time.Time {
	return timeFromMsec(d.msec)
}

func (d *dateObject) timeUTC() time.Time {
	return timeFromMsec(d.msec).In(time.UTC)
}
