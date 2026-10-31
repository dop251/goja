package goja

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja/unistring"
)

const (
	intlDateFull = iota
	intlDateLong
	intlDateMedium
	intlDateShort
)

type intlDateTimeFormatObject struct {
	baseObject
	locale          string
	tag             *intlTag
	dataLocale      string
	cal             *intlDateLocaleData
	timeZone        string
	location        *time.Location
	calendar        string
	numberingSystem string
	hourCycle       string
	hour12          bool
	hour12Set       bool
	hasTime         bool

	weekday, era, year, month, day, dayPeriod, hour, minute, second, fractionalSecond, timeZoneName string
	fractionalSecondDigits                                                                          int
	dateStyle, timeStyle                                                                            string
	pattern                                                                                         string

	bound *Object
}

func (r *Runtime) getIntlDateTimeFormat() *Object {
	ret := r.global.IntlDateTimeFormat
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlDateTimeFormat = ret
		proto := r.getIntlDateTimeFormatPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			return r.builtin_newIntlDateTimeFormat(call.Arguments, r.getIntlDateTimeFormatPrototype())
		}, r.wrapNativeConstruct(r.builtin_newIntlDateTimeFormat, ret, proto), proto, "DateTimeFormat", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlDateTimeFormatSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlDateTimeFormatPrototype() *Object {
	ret := r.global.IntlDateTimeFormatPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlDateTimeFormatPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlDateTimeFormat(), true, false, true)
		o.setOwnStr("format", &valueProperty{
			getterFunc:   r.newNativeFunc(r.intlDateTimeFormatGetFormat, "get format", 0),
			accessor:     true,
			configurable: true,
		}, false)
		o._putProp("formatToParts", r.newNativeFunc(r.intlDateTimeFormatProtoFormatToParts, "formatToParts", 1), true, false, true)
		o._putProp("formatRange", r.newNativeFunc(r.intlDateTimeFormatProtoFormatRange, "formatRange", 2), true, false, true)
		o._putProp("formatRangeToParts", r.newNativeFunc(r.intlDateTimeFormatProtoFormatRangeToParts, "formatRangeToParts", 2), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlDateTimeFormatProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.DateTimeFormat"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlDateTimeFormatThis(call FunctionCall) *intlDateTimeFormatObject {
	o, ok := r.toObject(call.This).self.(*intlDateTimeFormatObject)
	if !ok {
		panic(r.NewTypeError("Intl.DateTimeFormat method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlDateTimeFormat(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)

	cal := "gregory"
	calendarSet := false
	numberingSystemSet := false
	hourCycleSet := false
	numberingSystem := ""
	timeZone := ""
	hour12 := false
	hour12Set := false
	hourCycle := ""
	weekday, era, year, month, day, dayPeriod := "", "", "", "", "", ""
	hour, minute, second, fractionalSecond, timeZoneName := "", "", "", "", ""
	fractionalSecondDigits := 0
	dateStyle, timeStyle := "", ""
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "calendar", "string", nil, _undefined); !IsUndefined(v) {
			c := strings.ToLower(v.String())
			if !intlIsUnicodeExtensionValue(c) {
				panic(r.intlRangeError("Invalid calendar"))
			}
			if a, ok := intlCalendarAlias[c]; ok {
				c = a
			}
			if c == "islamic" {
				c = "islamic-civil"
			}
			if intlContains(intlCalendarValues, c) {
				cal = c
			}
			calendarSet = true
		}
		if v := r.intlGetOption(options, "numberingSystem", "string", nil, _undefined); !IsUndefined(v) {
			ns := strings.ToLower(v.String())
			if !intlIsUnicodeExtensionValue(ns) || !intlValidNumberingSystem(ns) {
				panic(r.intlRangeError("Invalid numberingSystem"))
			}
			numberingSystem = ns
			numberingSystemSet = true
		}
		if v := r.intlGetOption(options, "hour12", "boolean", nil, _undefined); !IsUndefined(v) {
			hour12 = v.ToBoolean()
			hour12Set = true
		}
		if v := r.intlGetOption(options, "hourCycle", "string", []string{"h11", "h12", "h23", "h24"}, _undefined); !IsUndefined(v) {
			hourCycle = v.String()
			hourCycleSet = true
		}
		if hour12Set && hourCycle != "" {
			panic(r.NewTypeError("hour12 and hourCycle can't both be set"))
		}
		if v := r.intlGetOption(options, "timeZone", "string", nil, _undefined); !IsUndefined(v) {
			timeZone = v.String()
		}
		weekday = r.intlComponentOption(options, "weekday", []string{"long", "short", "narrow"})
		era = r.intlComponentOption(options, "era", []string{"long", "short", "narrow"})
		year = r.intlComponentOption(options, "year", []string{"numeric", "2-digit"})
		month = r.intlComponentOption(options, "month", []string{"numeric", "2-digit", "long", "short", "narrow"})
		day = r.intlComponentOption(options, "day", []string{"numeric", "2-digit"})
		dayPeriod = r.intlComponentOption(options, "dayPeriod", []string{"narrow", "short", "long"})
		hour = r.intlComponentOption(options, "hour", []string{"numeric", "2-digit"})
		minute = r.intlComponentOption(options, "minute", []string{"numeric", "2-digit"})
		second = r.intlComponentOption(options, "second", []string{"numeric", "2-digit"})
		if v := nilSafe(options.self.getStr("fractionalSecondDigits", nil)); !IsUndefined(v) {
			n := int(v.ToInteger())
			if n < 1 || n > 3 {
				panic(r.intlRangeError("Invalid fractionalSecondDigits"))
			}
			fractionalSecond = strings.Repeat("S", n)
			fractionalSecondDigits = n
		}
		if v := r.intlGetOption(options, "timeZoneName", "string", []string{"short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"}, _undefined); !IsUndefined(v) {
			timeZoneName = v.String()
		}
		r.intlGetOption(options, "formatMatcher", "string", []string{"basic", "best fit"}, _undefined)
		if v := r.intlGetOption(options, "dateStyle", "string", []string{"full", "long", "medium", "short"}, _undefined); !IsUndefined(v) {
			dateStyle = v.String()
		}
		if v := r.intlGetOption(options, "timeStyle", "string", []string{"full", "long", "medium", "short"}, _undefined); !IsUndefined(v) {
			timeStyle = v.String()
		}
		if dateStyle != "" || timeStyle != "" {
			if weekday != "" || era != "" || year != "" || month != "" || day != "" || dayPeriod != "" ||
				hour != "" || minute != "" || second != "" || fractionalSecond != "" || timeZoneName != "" {
				panic(r.NewTypeError("dateStyle/timeStyle cannot be combined with component options"))
			}
		}
	}

	hasTime := hour != "" || minute != "" || second != "" || fractionalSecond != "" || dayPeriod != "" || timeZoneName != ""
	if dateStyle == "" && timeStyle == "" && !hasTime && weekday == "" && era == "" && year == "" && month == "" && day == "" {
		year, month, day = "numeric", "numeric", "numeric"
	}

	baseLocale, tag := r.intlResolveServiceLocale(requested, nil, options)
	locale := baseLocale
	dataLocale := "en"
	if key, ok := intlDataLocale(tag); ok {
		dataLocale = key
	}
	calData := intlDateLocales[dataLocale]
	if calData == nil {
		calData = intlDateLocales["en"]
	}
	if v, ok := tag.uKw["ca"]; ok && cal == "gregory" && intlContains(intlCalendarValues, v) {
		cal = v
	}
	if numberingSystem == "" {
		if v, ok := tag.uKw["nu"]; ok && v != "" {
			numberingSystem = v
		} else {
			numberingSystem = intlNumberLocaleFor(dataLocale).DefaultNS
		}
	}
	if timeZone == "" {
		timeZone = "UTC"
	}
	loc, tzName, err := intlLoadTimeZone(timeZone)
	if err != nil {
		panic(r.intlRangeError("Invalid time zone: " + timeZone))
	}
	timeZone = tzName

	if hourCycle == "" {
		if v, ok := tag.uKw["hc"]; ok && v != "" {
			hourCycle = v
		} else {
			hourCycle = intlDefaultHourCycle(tag)
		}
	}
	if hour12Set {
		if hour12 {
			if hourCycle == "h23" || hourCycle == "h24" {
				hourCycle = "h12"
			}
		} else {
			if hourCycle == "h11" || hourCycle == "h12" {
				hourCycle = "h23"
			}
		}
	}
	hour12 = hourCycle == "h11" || hourCycle == "h12"

	var ext []string
	if v, ok := tag.uKw["ca"]; ok && calendarSet == false && v == cal {
		ext = append(ext, "ca-"+v)
	} else if v, ok := tag.uKw["ca"]; ok && calendarSet && v == cal {
		ext = append(ext, "ca-"+v)
	}
	if v, ok := tag.uKw["nu"]; ok && intlValidNumberingSystem(v) && v == numberingSystem {
		ext = append(ext, "nu-"+v)
	}
	if v, ok := tag.uKw["hc"]; ok && v == hourCycle {
		if hourCycle == "h11" || hourCycle == "h12" || hourCycle == "h23" || hourCycle == "h24" {
			ext = append(ext, "hc-"+v)
		}
	}
	if len(ext) > 0 {
		locale = baseLocale + "-u-" + strings.Join(ext, "-")
	}
	_, _, _ = calendarSet, numberingSystemSet, hourCycleSet

	o := &Object{runtime: r}
	obj := &intlDateTimeFormatObject{
		locale: locale, tag: tag, dataLocale: dataLocale, cal: calData,
		timeZone: timeZone, location: loc, calendar: cal, numberingSystem: numberingSystem,
		hourCycle: hourCycle, hour12: hour12, hour12Set: hour12Set, hasTime: hasTime,
		weekday: weekday, era: era, year: year, month: month, day: day, dayPeriod: dayPeriod,
		hour: hour, minute: minute, second: second, fractionalSecond: fractionalSecond,
		timeZoneName: timeZoneName, dateStyle: dateStyle, timeStyle: timeStyle,
		fractionalSecondDigits: fractionalSecondDigits,
	}
	obj.pattern = obj.resolvePattern()
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (r *Runtime) intlComponentOption(options *Object, name string, allowed []string) string {
	v := nilSafe(options.self.getStr(unistring.NewFromString(name), nil))
	if IsUndefined(v) {
		return ""
	}
	s := v.toString().String()
	for _, a := range allowed {
		if s == a {
			return s
		}
	}
	panic(r.intlRangeError("Invalid value for " + name + ": " + s))
}

func intlDefaultHourCycle(tag *intlTag) string {
	if tag.region == "" {
		tag = intlMaximize(tag)
	}
	region := tag.region
	if region == "" {
		region = "001"
	}
	if hc, ok := intlHourCyclePref[region]; ok {
		return hc
	}
	if hc, ok := intlHourCyclePref["001"]; ok {
		return hc
	}
	return "h23"
}

func intlLoadTimeZone(name string) (*time.Location, string, error) {
	if name == "UTC" || name == "GMT" {
		return time.UTC, "UTC", nil
	}
	if name == "Etc/UTC" {
		return time.UTC, "Etc/UTC", nil
	}
	if name == "Etc/GMT" {
		return time.FixedZone("Etc/GMT", 0), "Etc/GMT", nil
	}
	// Fixed offsets: ±HH:MM or ±HHMM.
	if len(name) >= 3 && (name[0] == '+' || name[0] == '-') {
		sign := 1
		if name[0] == '-' {
			sign = -1
		}
		body := name[1:]
		if len(body) != 2 && len(body) != 5 {
			return nil, "", strconv.ErrSyntax
		}
		if len(body) == 5 && body[2] != ':' {
			return nil, "", strconv.ErrSyntax
		}
		rest := strings.Replace(body, ":", "", 1)
		if len(rest) == 2 || len(rest) == 4 {
			h, err1 := strconv.Atoi(rest[:2])
			m := 0
			var err2 error
			if len(rest) == 4 {
				m, err2 = strconv.Atoi(rest[2:])
			}
			if err1 == nil && err2 == nil && h < 24 && m < 60 {
				off := sign * (h*3600 + m*60)
				return time.FixedZone(name, off), name, nil
			}
		}
		return nil, "", strconv.ErrSyntax
	}
	if strings.Contains(name, "/") || name == "UTC" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc, loc.String(), nil
		}
	}
	return nil, "", strconv.ErrSyntax
}

func (d *intlDateTimeFormatObject) resolvePattern() string {
	var datePart, timePart string
	if d.dateStyle != "" || d.timeStyle != "" {
		if d.dateStyle != "" {
			datePart = d.cal.DateFormats[intlStyleIndex(d.dateStyle)]
		}
		if d.timeStyle != "" {
			timePart = d.cal.TimeFormats[intlStyleIndex(d.timeStyle)]
		}
		if datePart != "" && timePart != "" {
			idx := intlDateMedium
			if d.dateStyle != "" && d.timeStyle != "" {
				di := intlStyleIndex(d.dateStyle)
				ti := intlStyleIndex(d.timeStyle)
				if ti < di {
					idx = ti
				} else {
					idx = di
				}
			} else if d.dateStyle != "" {
				idx = intlStyleIndex(d.dateStyle)
			} else {
				idx = intlStyleIndex(d.timeStyle)
			}
			return strings.ReplaceAll(strings.ReplaceAll(d.cal.DateTimeFormats[idx], "{1}", datePart), "{0}", timePart)
		}
		if datePart != "" {
			return datePart
		}
		return timePart
	}

	var dateFields []string
	if d.era != "" {
		dateFields = append(dateFields, intlEraSkeleton(d.era))
	}
	if d.year != "" {
		if d.year == "2-digit" {
			dateFields = append(dateFields, "yy")
		} else {
			dateFields = append(dateFields, "y")
		}
	}
	if d.month != "" {
		dateFields = append(dateFields, intlMonthSkeleton(d.month))
	}
	if d.weekday != "" {
		dateFields = append(dateFields, intlWeekdaySkeleton(d.weekday))
	}
	if d.day != "" {
		if d.day == "2-digit" {
			dateFields = append(dateFields, "dd")
		} else {
			dateFields = append(dateFields, "d")
		}
	}

	hourLetter := "h"
	if !d.hour12 {
		hourLetter = "H"
	}
	var timeFields []string
	if d.dayPeriod != "" {
		timeFields = append(timeFields, "a")
	}
	if d.hour != "" {
		if d.hour == "2-digit" {
			timeFields = append(timeFields, hourLetter+hourLetter)
		} else {
			timeFields = append(timeFields, hourLetter)
		}
	}
	if d.minute != "" {
		timeFields = append(timeFields, "mm")
	}
	if d.second != "" {
		timeFields = append(timeFields, "ss")
	}
	if d.fractionalSecond != "" {
		timeFields = append(timeFields, strings.Repeat("S", len(d.fractionalSecond)))
	}
	if d.timeZoneName != "" {
		timeFields = append(timeFields, intlTimeZoneNameSkeleton(d.timeZoneName))
	}

	dateSkel := strings.Join(dateFields, "")
	timeSkel := strings.Join(timeFields, "")
	widths := d.requestedWidths()

	if dateSkel != "" && timeSkel != "" {
		p := d.lookupPattern(dateSkel + timeSkel)
		if p == "" {
			p = d.lookupPattern(timeSkel + dateSkel)
		}
		if p != "" {
			return d.appendMissingFields(intlAdjustPattern(p, widths), dateSkel+timeSkel)
		}
		dp := d.lookupPattern(dateSkel)
		tp := d.lookupPattern(timeSkel)
		if dp == "" {
			dp = d.fallbackDatePattern()
		}
		if tp == "" {
			tp = d.fallbackTimePattern()
		}
		return strings.ReplaceAll(strings.ReplaceAll(d.cal.DateTimeFormats[intlDateMedium], "{1}", dp), "{0}", tp)
	}
	if dateSkel != "" {
		if p := d.lookupPattern(dateSkel); p != "" {
			return d.appendMissingFields(intlAdjustPattern(p, widths), dateSkel)
		}
		return d.fallbackDatePattern()
	}
	if p := d.lookupPattern(timeSkel); p != "" {
		return d.appendMissingFields(intlAdjustPattern(p, widths), timeSkel)
	}
	return d.fallbackTimePattern()
}

func intlStyleIndex(style string) int {
	switch style {
	case "full":
		return intlDateFull
	case "long":
		return intlDateLong
	case "medium":
		return intlDateMedium
	}
	return intlDateShort
}

func intlEraSkeleton(v string) string {
	switch v {
	case "long":
		return "GGGG"
	case "narrow":
		return "GGGGG"
	}
	return "G"
}

func intlMonthSkeleton(v string) string {
	switch v {
	case "2-digit":
		return "MM"
	case "long":
		return "MMMM"
	case "short":
		return "MMM"
	case "narrow":
		return "MMMMM"
	}
	return "M"
}

func intlWeekdaySkeleton(v string) string {
	switch v {
	case "long":
		return "EEEE"
	case "narrow":
		return "EEEEE"
	}
	return "E"
}

func intlTimeZoneNameSkeleton(v string) string {
	switch v {
	case "long":
		return "zzzz"
	case "shortOffset":
		return "O"
	case "longOffset":
		return "OOOO"
	case "shortGeneric":
		return "v"
	case "longGeneric":
		return "vvvv"
	}
	return "z"
}

func (d *intlDateTimeFormatObject) lookupPattern(skel string) string {
	if p, ok := d.cal.Available[skel]; ok {
		return p
	}
	if p, ok := intlBestAvailablePattern(d.cal.Available, skel); ok {
		return p
	}
	return ""
}

// intlSkeletonFields returns the requested field widths keyed by field letter.
func intlSkeletonFields(skel string) map[byte]int {
	res := map[byte]int{}
	for i := 0; i < len(skel); i++ {
		c := skel[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			continue
		}
		res[c]++
	}
	return res
}

func intlSkeletonSet(skel string) map[byte]int { return intlSkeletonFields(skel) }

func intlWidthDistance(req, cand int) int {
	if req == cand {
		return 0
	}
	if req < 3 && cand < 3 {
		return 0
	}
	d := req - cand
	if d < 0 {
		d = -d
	}
	return d
}

func intlSameFields(a, b map[byte]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// intlBestAvailablePattern chooses the CLDR pattern whose field set matches the
// requested skeleton most closely.
func intlBestAvailablePattern(avail map[string]string, skel string) (string, bool) {
	req := intlSkeletonFields(skel)
	keys := make([]string, 0, len(avail))
	for k := range avail {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best := ""
	bestDist := 1 << 30
	for _, key := range keys {
		f := intlSkeletonFields(key)
		if !intlSameFields(req, f) {
			continue
		}
		dist := 0
		for k, rv := range req {
			dist += intlWidthDistance(rv, f[k])
		}
		if dist < bestDist {
			bestDist = dist
			best = avail[key]
			if dist == 0 {
				return best, true
			}
		}
	}
	if best != "" {
		return best, true
	}
	// Fall back to a subset match: pick the pattern sharing the most fields.
	bestScore := -1
	for _, key := range keys {
		f := intlSkeletonFields(key)
		score := 0
		for k := range req {
			if _, ok := f[k]; ok {
				score++
			} else {
				score -= 2
			}
		}
		for k := range f {
			if _, ok := req[k]; !ok {
				score -= 1
			}
		}
		if score > bestScore {
			bestScore = score
			best = avail[key]
		}
	}
	return best, best != ""
}

func (d *intlDateTimeFormatObject) fallbackDatePattern() string {
	var parts []string
	if d.era != "" {
		parts = append(parts, intlEraSkeleton(d.era))
	}
	if d.year != "" {
		parts = append(parts, "y")
	}
	if d.month != "" {
		parts = append(parts, intlMonthSkeleton(d.month))
	}
	if d.day != "" {
		parts = append(parts, "d")
	}
	if d.weekday != "" {
		parts = append(parts, intlWeekdaySkeleton(d.weekday))
	}
	return strings.Join(parts, " ")
}

func (d *intlDateTimeFormatObject) fallbackTimePattern() string {
	var parts []string
	if d.hour != "" {
		if d.hour12 {
			parts = append(parts, "h")
		} else {
			parts = append(parts, "H")
		}
	}
	if d.minute != "" {
		parts = append(parts, "mm")
	}
	if d.second != "" {
		parts = append(parts, "ss")
	}
	p := strings.Join(parts, ":")
	if d.dayPeriod != "" || d.hour12 {
		p += " a"
	}
	return p
}

// appendMissingFields adds requested fields that the chosen pattern lacks.
func (d *intlDateTimeFormatObject) appendMissingFields(pattern, skel string) string {
	has := map[byte]bool{}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case 'L':
			c = 'M'
		case 'K', 'k':
			c = 'h'
		case 'e', 'c':
			c = 'E'
		case 'Y', 'u':
			c = 'y'
		}
		has[c] = true
	}
	if d.dayPeriod != "" && !has['a'] && !has['B'] {
		pattern += " " + strings.Repeat("a", intlDayPeriodWidth(d.dayPeriod))
	}
	if d.fractionalSecond != "" && !has['S'] {
		pattern += "." + d.fractionalSecond
	}
	if d.timeZoneName != "" && !has['z'] {
		pattern += " " + intlTimeZoneNameSkeleton(d.timeZoneName)
	}
	return pattern
}

func intlDayPeriodWidth(v string) int {
	switch v {
	case "long":
		return 4
	case "narrow":
		return 5
	}
	return 1
}

func (d *intlDateTimeFormatObject) requestedWidths() map[byte]string {
	w := map[byte]string{}
	if d.era != "" {
		w['G'] = intlEraSkeleton(d.era)
	}
	if d.year != "" {
		if d.year == "2-digit" {
			w['y'] = "yy"
		} else {
			w['y'] = "y"
		}
	}
	if d.month != "" {
		w['M'] = intlMonthSkeleton(d.month)
	}
	if d.day != "" {
		if d.day == "2-digit" {
			w['d'] = "dd"
		} else {
			w['d'] = "d"
		}
	}
	if d.weekday != "" {
		w['E'] = intlWeekdaySkeleton(d.weekday)
	}
	if d.hour != "" {
		letter := "h"
		if !d.hour12 {
			letter = "H"
		}
		if d.hour == "2-digit" {
			w['h'] = letter + letter
			w['H'] = letter + letter
		} else {
			w['h'] = letter
			w['H'] = letter
		}
	}
	if d.minute != "" {
		w['m'] = "mm"
	}
	if d.second != "" {
		w['s'] = "ss"
	}
	if d.fractionalSecond != "" {
		w['S'] = d.fractionalSecond
	}
	if d.timeZoneName != "" {
		w['z'] = intlTimeZoneNameSkeleton(d.timeZoneName)
	}
	return w
}

// intlAdjustPattern applies the requested field widths to a CLDR pattern and
// removes fields that were not requested.
func intlAdjustPattern(pattern string, widths map[byte]string) string {
	var sb strings.Builder
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\'' {
			j := i + 1
			for j < len(pattern) && pattern[j] != '\'' {
				j++
			}
			if j < len(pattern) {
				sb.WriteString(pattern[i : j+1])
				i = j
			} else {
				sb.WriteByte(c)
			}
			continue
		}
		if !intlIsDateField(c) {
			sb.WriteByte(c)
			continue
		}
		j := i
		for j < len(pattern) && pattern[j] == c {
			j++
		}
		run := j - i
		i = j - 1
		key := c
		switch c {
		case 'L':
			key = 'M'
		case 'K', 'k':
			key = 'h'
		case 'e', 'c':
			key = 'E'
		case 'Y', 'u':
			key = 'y'
		}
		if w, ok := widths[key]; ok {
			if key == 'M' && strings.HasPrefix(pattern[j:], "月") {
				// CJK month names carry the 月 suffix; keep the numeric field.
				w = "M"
			}
			sb.WriteString(w)
		} else if key == 'h' || key == 'H' || key == 'm' || key == 's' || key == 'S' || key == 'G' || key == 'M' || key == 'd' || key == 'E' || key == 'y' || key == 'z' || key == 'a' {
			// Drop unrequested fields, except the day period that comes with
			// a 12-hour time pattern.
			if key == 'a' {
				sb.WriteString(strings.Repeat(string(c), run))
			}
		} else {
			sb.WriteString(strings.Repeat(string(c), run))
		}
	}
	res := sb.String()
	return intlCleanupPattern(res)
}

func intlCleanupPattern(p string) string {
	res := p
	for {
		prev := res
		res = strings.ReplaceAll(res, ", ,", ",")
		res = strings.ReplaceAll(res, ",,", ",")
		res = strings.ReplaceAll(res, "  ", " ")
		res = strings.ReplaceAll(res, ", ", ", ")
		res = strings.TrimPrefix(res, ", ")
		res = strings.TrimPrefix(res, ", ")
		res = strings.TrimSuffix(res, ", ")
		res = strings.TrimSuffix(res, " ")
		res = strings.TrimPrefix(res, " ")
		if res == prev {
			break
		}
	}
	return res
}

// ---------- formatting ----------

func (d *intlDateTimeFormatObject) formatDate(value float64) []intlStringPart {
	if math.IsNaN(value) {
		return []intlStringPart{{"literal", "Invalid Date"}}
	}
	ms := int64(value)
	sec := ms / 1000
	nsec := (ms % 1000) * int64(time.Millisecond)
	t := time.Unix(sec, nsec).In(d.location)
	return d.formatTime(t)
}

func (d *intlDateTimeFormatObject) formatTime(t time.Time) []intlStringPart {
	digits := intlNumberingDigits(d.numberingSystem)
	pattern := d.pattern
	var parts []intlStringPart
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			parts = append(parts, intlStringPart{"literal", lit.String()})
			lit.Reset()
		}
	}
	add := func(typ, val string) {
		flush()
		parts = append(parts, intlStringPart{typ, val})
	}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\'' {
			// Quoted literal.
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				lit.WriteByte('\'')
				i++
				continue
			}
			i++
			for i < len(pattern) && pattern[i] != '\'' {
				lit.WriteByte(pattern[i])
				i++
			}
			continue
		}
		if !intlIsDateField(c) {
			lit.WriteByte(c)
			continue
		}
		j := i
		for j < len(pattern) && pattern[j] == c {
			j++
		}
		n := j - i
		i = j - 1
		val, typ := d.renderField(c, n, t, digits)
		if typ == "" {
			continue
		}
		add(typ, val)
	}
	flush()
	return parts
}

func intlIsDateField(c byte) bool {
	switch c {
	case 'G', 'y', 'Y', 'u', 'M', 'L', 'd', 'E', 'e', 'c', 'a', 'b', 'B', 'h', 'H', 'K', 'k', 'm', 's', 'S', 'z', 'Z', 'O', 'v', 'V', 'X', 'x', 'Q', 'q', 'D', 'F', 'w', 'W':
		return true
	}
	return false
}

func (d *intlDateTimeFormatObject) renderField(c byte, n int, t time.Time, digits string) (string, string) {
	pad := func(v, width int) string {
		s := strconv.Itoa(v)
		for len(s) < width {
			s = "0" + s
		}
		return intlMapDigits(s, digits)
	}
	switch c {
	case 'G':
		idx := 1
		if t.Year() <= 0 {
			idx = 0
		}
		var names [2]string
		switch {
		case n >= 5:
			names = d.cal.ErasNarrow
		case n == 4:
			names = d.cal.ErasWide
		default:
			names = d.cal.ErasAbbr
		}
		return names[idx], "era"
	case 'y', 'Y', 'u':
		y := t.Year()
		if y <= 0 {
			y = 1 - y
		}
		if c == 'Y' || c == 'u' {
			y = t.Year()
		}
		if n == 2 {
			return intlMapDigits(strconv.Itoa(y%100), digits), "year"
		}
		if n >= 4 {
			return pad(y, n), "year"
		}
		return intlMapDigits(strconv.Itoa(y), digits), "year"
	case 'M', 'L':
		m := int(t.Month())
		switch {
		case n >= 5:
			v := d.cal.MonthsNarrow
			if m-1 < len(v) {
				return v[m-1], "month"
			}
		case n == 4:
			v := d.cal.MonthsWide
			if m-1 < len(v) {
				return v[m-1], "month"
			}
		case n == 3:
			v := d.cal.MonthsAbbr
			if m-1 < len(v) {
				return v[m-1], "month"
			}
		}
		if n == 2 {
			return pad(m, 2), "month"
		}
		return intlMapDigits(strconv.Itoa(m), digits), "month"
	case 'd':
		if n == 2 {
			return pad(t.Day(), 2), "day"
		}
		return intlMapDigits(strconv.Itoa(t.Day()), digits), "day"
	case 'E', 'e', 'c':
		wd := int(t.Weekday())
		var v []string
		switch {
		case n >= 5:
			v = d.cal.DaysNarrow
		case n == 4:
			v = d.cal.DaysWide
		case n == 6:
			v = d.cal.DaysShort
		default:
			v = d.cal.DaysAbbr
		}
		if wd < len(v) {
			return v[wd], "weekday"
		}
		return "", ""
	case 'a', 'b', 'B':
		pm := t.Hour() >= 12
		var names [2]string
		switch {
		case n >= 5:
			names = d.cal.DayPeriodsNarrow
		case n == 4:
			names = d.cal.DayPeriodsWide
		default:
			names = d.cal.DayPeriodsAbbr
		}
		if pm {
			return names[1], "dayPeriod"
		}
		return names[0], "dayPeriod"
	case 'h':
		h := t.Hour() % 12
		if h == 0 {
			h = 12
		}
		if n == 2 {
			return pad(h, 2), "hour"
		}
		return intlMapDigits(strconv.Itoa(h), digits), "hour"
	case 'H':
		if n == 2 {
			return pad(t.Hour(), 2), "hour"
		}
		return intlMapDigits(strconv.Itoa(t.Hour()), digits), "hour"
	case 'K':
		h := t.Hour() % 12
		if n == 2 {
			return pad(h, 2), "hour"
		}
		return intlMapDigits(strconv.Itoa(h), digits), "hour"
	case 'k':
		h := t.Hour()
		if h == 0 {
			h = 24
		}
		if n == 2 {
			return pad(h, 2), "hour"
		}
		return intlMapDigits(strconv.Itoa(h), digits), "hour"
	case 'm':
		if n == 2 {
			return pad(t.Minute(), 2), "minute"
		}
		return intlMapDigits(strconv.Itoa(t.Minute()), digits), "minute"
	case 's':
		if n == 2 {
			return pad(t.Second(), 2), "second"
		}
		return intlMapDigits(strconv.Itoa(t.Second()), digits), "second"
	case 'S':
		frac := t.Nanosecond() / 1000000
		s := pad(frac, 3)
		if n < 3 {
			s = s[:n]
		}
		return intlMapDigits(s, digits), "fractionalSecond"
	case 'z', 'Z', 'O', 'v', 'V', 'X', 'x':
		return d.formatTimeZone(t, n, c), "timeZoneName"
	}
	return "", ""
}

func intlAbbrevZoneName(name string) string {
	parts := strings.Split(name, "/")
	abbr := parts[len(parts)-1]
	abbr = strings.ReplaceAll(abbr, "_", " ")
	if len(abbr) > 3 {
		// Use the zone abbreviation from Go where possible.
		return abbr
	}
	return strings.ToUpper(abbr)
}

func (d *intlDateTimeFormatObject) formatTimeZone(t time.Time, n int, c byte) string {
	name, offset := t.Zone()
	// Prefer Go's abbreviation when it looks like a real one.
	if len(name) <= 5 && name != "" && (c == 'z' || c == 'v' || c == 'Z') {
		return name
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	h := offset / 3600
	m := (offset % 3600) / 60
	off := "GMT"
	if offset != 0 {
		off = "GMT" + sign + strconv.Itoa(h)
		if m != 0 {
			off += ":" + strconv.Itoa(m/10) + strconv.Itoa(m%10)
		}
	}
	if d.timeZone == "UTC" && (c == 'z' || c == 'v') {
		if n >= 4 {
			return "Coordinated Universal Time"
		}
		return "UTC"
	}
	if n >= 4 && intlAbbrevZoneName(d.timeZone) != "" {
		return strings.ReplaceAll(intlAbbrevZoneName(d.timeZone), "/", " ")
	}
	return off
}

func (r *Runtime) intlDateTimeFormatGetFormat(call FunctionCall) Value {
	d := r.intlDateTimeFormatThis(call)
	if d.bound == nil {
		d.bound = r.newNativeFunc(func(call FunctionCall) Value {
			return intlStr(intlPartsToString(d.formatDate(r.intlFormatArgMillis(call.Argument(0)))))
		}, "", 1)
	}
	return d.bound
}

func (r *Runtime) intlDateTimeFormatProtoFormatToParts(call FunctionCall) Value {
	d := r.intlDateTimeFormatThis(call)
	return r.intlPartsArray(d.formatDate(r.intlFormatArgMillis(call.Argument(0))))
}

func (r *Runtime) intlDateTimeFormatProtoFormatRange(call FunctionCall) Value {
	d := r.intlDateTimeFormatThis(call)
	a := r.intlFormatArgMillis(call.Argument(0))
	b := r.intlFormatArgMillis(call.Argument(1))
	if math.IsNaN(a) || math.IsInf(a, 0) || math.IsNaN(b) || math.IsInf(b, 0) {
		panic(r.intlRangeError("Invalid range"))
	}
	if a == b {
		return intlStr(intlPartsToString(d.formatDate(a)))
	}
	return intlStr(intlPartsToString(d.formatDate(a)) + " – " + intlPartsToString(d.formatDate(b)))
}

func (r *Runtime) intlDateTimeFormatProtoFormatRangeToParts(call FunctionCall) Value {
	d := r.intlDateTimeFormatThis(call)
	a := r.intlFormatArgMillis(call.Argument(0))
	b := r.intlFormatArgMillis(call.Argument(1))
	if math.IsNaN(a) || math.IsInf(a, 0) || math.IsNaN(b) || math.IsInf(b, 0) {
		panic(r.intlRangeError("Invalid range"))
	}
	var vals []Value
	add := func(typ, val, source string) {
		obj := r.newBaseObject(r.global.ObjectPrototype, classObject)
		obj.setOwnStr("type", asciiString(typ), false)
		obj.setOwnStr("value", intlStr(val), false)
		obj.setOwnStr("source", asciiString(source), false)
		vals = append(vals, obj.val)
	}
	for _, p := range d.formatDate(a) {
		add(p.typ, p.value, "startRange")
	}
	add("literal", " – ", "shared")
	for _, p := range d.formatDate(b) {
		add(p.typ, p.value, "endRange")
	}
	return r.newArrayValues(vals)
}

func (r *Runtime) intlDateTimeFormatProtoResolvedOptions(call FunctionCall) Value {
	d := r.intlDateTimeFormatThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(d.locale), false)
	o.setOwnStr("calendar", intlStr(d.calendar), false)
	o.setOwnStr("numberingSystem", intlStr(d.numberingSystem), false)
	o.setOwnStr("timeZone", intlStr(d.timeZone), false)
	if d.hasTime || d.timeStyle != "" {
		o.setOwnStr("hourCycle", intlStr(d.hourCycle), false)
		o.setOwnStr("hour12", r.ToValue(d.hour12), false)
	}
	if d.dateStyle != "" || d.timeStyle != "" {
		if d.dateStyle != "" {
			o.setOwnStr("dateStyle", intlStr(d.dateStyle), false)
		}
		if d.timeStyle != "" {
			o.setOwnStr("timeStyle", intlStr(d.timeStyle), false)
		}
		return o.val
	}
	if d.weekday != "" {
		o.setOwnStr("weekday", intlStr(d.weekday), false)
	}
	if d.era != "" {
		o.setOwnStr("era", intlStr(d.era), false)
	}
	if d.year != "" {
		o.setOwnStr("year", intlStr(d.year), false)
	}
	if d.month != "" {
		o.setOwnStr("month", intlStr(d.month), false)
	}
	if d.day != "" {
		o.setOwnStr("day", intlStr(d.day), false)
	}
	if d.dayPeriod != "" {
		o.setOwnStr("dayPeriod", intlStr(d.dayPeriod), false)
	}
	if d.hour != "" {
		o.setOwnStr("hour", intlStr(d.hour), false)
	}
	if d.minute != "" {
		o.setOwnStr("minute", intlStr(d.minute), false)
	}
	if d.second != "" {
		o.setOwnStr("second", intlStr(d.second), false)
	}
	if d.fractionalSecond != "" {
		o.setOwnStr("fractionalSecondDigits", intToValue(int64(d.fractionalSecondDigits)), false)
	}
	if d.timeZoneName != "" {
		o.setOwnStr("timeZoneName", intlStr(d.timeZoneName), false)
	}
	return o.val
}

func (r *Runtime) intlDateTimeFormatSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, func(tag *intlTag) bool {
		if _, ok := intlDateLocales[tag.baseName()]; ok {
			return true
		}
		return intlHasLocaleData(tag)
	})
}

var _ = time.Now
