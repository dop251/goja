package goja

import (
	"strings"
)

type date struct {
	year, month, day     int
	hour, min, sec, msec int
	timeZoneOffset       int // time zone offset in minutes
	isLocal              bool
}

func skip(sp *string, c byte) bool {
	s := *sp
	if len(s) > 0 && s[0] == c {
		*sp = s[1:]
		return true
	}
	return false
}

func skipUntil(sp *string, stopList string) {
	s := *sp
	for len(s) > 0 && !strings.ContainsRune(stopList, rune(s[0])) {
		s = s[1:]
	}
	*sp = s
}

func match(sp *string, lower string) bool {
	s := *sp
	if len(s) < len(lower) {
		return false
	}
	for i := 0; i < len(lower); i++ {
		c1 := s[i]
		c2 := lower[i]
		if c1 != c2 {
			// switch to lower-case; 'a'-'A' is known to be a single bit
			c1 |= 'a' - 'A'
			if c1 != c2 || c1 < 'a' || c1 > 'z' {
				return false
			}
		}
	}
	*sp = s[len(lower):]
	return true
}

func getDigits(sp *string, minDigits, maxDigits int, val *int) bool {
	s := *sp
	var p, v int
	for p < len(s) && p < maxDigits && s[p] >= '0' && s[p] <= '9' {
		v = v*10 + int(s[p]-'0')
		p++
	}
	if p < minDigits {
		return false
	}
	*sp, *val = s[p:], v
	return true
}

func getMilliseconds(sp *string, val *int) {
	s := *sp
	mul, v := 100, 0
	if len(s) > 0 && (s[0] == '.' || s[0] == ',') {
		const P_START = 1
		p := P_START
		for p < len(s) && p-P_START < 9 && s[p] >= '0' && s[p] <= '9' {
			v += int(s[p]-'0') * mul
			mul /= 10
			p++
		}
		if p > P_START {
			// only consume the separator if digits are present
			*sp, *val = s[p:], v
		}
	}
}

// [+-]HH:mm or [+-]HHmm or Z
func getTimeZoneOffset(sp *string, strict bool, offset *int) bool {
	s := *sp
	if len(s) == 0 {
		return false
	}
	var hh, mm, v int
	sign, s := s[0], s[1:]
	if sign == '+' || sign == '-' {
		n := len(s)
		if !getDigits(&s, 1, 9, &hh) {
			return false
		}
		n -= len(s)
		if strict && n != 2 && n != 4 {
			return false
		}
		for n > 4 {
			n -= 2
			hh /= 100
		}
		if n > 2 {
			mm = hh % 100
			hh = hh / 100
		} else if skip(&s, ':') && !getDigits(&s, 2, 2, &mm) {
			return false
		}
		if hh > 23 || mm > 59 {
			return false
		}
		v = hh*60 + mm
		if sign == '-' {
			v = -v
		}
	} else if sign != 'Z' {
		return false
	}
	*sp, *offset = s, v
	return true
}

var tzAbbrs = []struct {
	nameLower string
	offset    int
}{
	{"gmt", 0},        // Greenwich Mean Time
	{"utc", 0},        // Coordinated Universal Time
	{"ut", 0},         // Universal Time
	{"z", 0},          // Zulu Time
	{"edt", -4 * 60},  // Eastern Daylight Time
	{"est", -5 * 60},  // Eastern Standard Time
	{"cdt", -5 * 60},  // Central Daylight Time
	{"cst", -6 * 60},  // Central Standard Time
	{"mdt", -6 * 60},  // Mountain Daylight Time
	{"mst", -7 * 60},  // Mountain Standard Time
	{"pdt", -7 * 60},  // Pacific Daylight Time
	{"pst", -8 * 60},  // Pacific Standard Time
	{"wet", +0 * 60},  // Western European Time
	{"west", +1 * 60}, // Western European Summer Time
	{"cet", +1 * 60},  // Central European Time
	{"cest", +2 * 60}, // Central European Summer Time
	{"eet", +2 * 60},  // Eastern European Time
	{"eest", +3 * 60}, // Eastern European Summer Time
}

func getTimeZoneAbbr(sp *string, offset *int) bool {
	for _, tzAbbr := range tzAbbrs {
		if match(sp, tzAbbr.nameLower) {
			*offset = tzAbbr.offset
			return true
		}
	}
	return false
}

var monthNamesLower = []string{
	"jan",
	"feb",
	"mar",
	"apr",
	"may",
	"jun",
	"jul",
	"aug",
	"sep",
	"oct",
	"nov",
	"dec",
}

func getMonth(sp *string, val *int) bool {
	for i, monthNameLower := range monthNamesLower {
		if match(sp, monthNameLower) {
			*val = i + 1
			return true
		}
	}
	return false
}

func parseDateISOString(s string) (d date, ok bool) {
	if len(s) == 0 {
		return
	}
	d.month = 1
	d.day = 1

	// year is either yyyy digits or [+-]yyyyyy
	sign := s[0]
	if sign == '-' || sign == '+' {
		s = s[1:]
		if !getDigits(&s, 6, 6, &d.year) {
			return
		}
		if sign == '-' {
			if d.year == 0 {
				// reject -000000
				return
			}
			d.year = -d.year
		}
	} else if !getDigits(&s, 4, 4, &d.year) {
		return
	}
	if skip(&s, '-') {
		if !getDigits(&s, 2, 2, &d.month) || d.month < 1 {
			return
		}
		if skip(&s, '-') {
			if !getDigits(&s, 2, 2, &d.day) || d.day < 1 {
				return
			}
		}
	}
	if skip(&s, 'T') {
		if !getDigits(&s, 2, 2, &d.hour) || !skip(&s, ':') || !getDigits(&s, 2, 2, &d.min) {
			return
		}
		if skip(&s, ':') {
			if !getDigits(&s, 2, 2, &d.sec) {
				return
			}
			getMilliseconds(&s, &d.msec)
		}
		d.isLocal = true
	}
	// parse the time zone offset if present
	if len(s) > 0 {
		if !getTimeZoneOffset(&s, true, &d.timeZoneOffset) {
			return
		}
		d.isLocal = false
	}
	// error if extraneous characters
	ok = len(s) == 0
	return
}

func parseDateOtherString(s string) (d date, ok bool) {
	d.year = 2001
	d.month = 1
	d.day = 1
	d.isLocal = true

	var nums [3]int
	var numIndex int
	var hasYear, hasMon, hasTime bool
	for {
		for len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
		n := len(s)
		if n == 0 {
			break
		}
		c := s[0]
		if c == '+' || c == '-' {
			if hasTime && getTimeZoneOffset(&s, false, &d.timeZoneOffset) {
				d.isLocal = false
			} else {
				s = s[1:]
				if getDigits(&s, 1, 9, &d.year) {
					if c == '-' {
						if d.year == 0 {
							return
						}
						d.year = -d.year
					}
					hasYear = true
				}
			}
		} else if val := 0; getDigits(&s, 1, 9, &val) {
			if skip(&s, ':') {
				// time part
				d.hour = val
				if !getDigits(&s, 1, 2, &d.min) {
					return
				}
				if skip(&s, ':') {
					if !getDigits(&s, 1, 2, &d.sec) {
						return
					}
					getMilliseconds(&s, &d.msec)
				}
				hasTime = true
			} else if n-len(s) > 2 {
				d.year = val
				hasYear = true
			} else if val < 1 || val > 31 {
				d.year = val
				if val < 100 {
					d.year += 1900
				}
				if val < 50 {
					d.year += 100
				}
				hasYear = true
			} else {
				if numIndex == 3 {
					return
				}
				nums[numIndex] = val
				numIndex++
			}
		} else if getMonth(&s, &d.month) {
			hasMon = true
			skipUntil(&s, "0123456789 -/(")
		} else if hasTime && match(&s, "pm") {
			if d.hour < 12 {
				d.hour += 12
			}
			continue
		} else if hasTime && match(&s, "am") {
			if d.hour == 12 {
				d.hour = 0
			}
			continue
		} else if getTimeZoneAbbr(&s, &d.timeZoneOffset) {
			if len(s) > 0 {
				if c := s[0]; (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					return
				}
			}
			d.isLocal = false
			continue
		} else if c == '(' {
			// skip parenthesized phrase
			level := 1
			s = s[1:]
			for len(s) > 0 && level != 0 {
				if s[0] == '(' {
					level++
				} else if s[0] == ')' {
					level--
				}
				s = s[1:]
			}
			if level > 0 {
				return
			}
		} else if c == ')' {
			return
		} else {
			if hasYear || hasMon || hasTime || numIndex > 0 {
				return
			}
			// skip a word
			skipUntil(&s, " -/(")
		}
		for len(s) > 0 && strings.ContainsRune("-/.,", rune(s[0])) {
			s = s[1:]
		}
	}
	n := numIndex
	if hasYear {
		n++
	}
	if hasMon {
		n++
	}
	if n > 3 {
		return
	}

	switch numIndex {
	case 0:
		if !hasYear {
			return
		}
	case 1:
		if hasMon {
			d.day = nums[0]
		} else {
			d.month = nums[0]
		}
	case 2:
		if hasYear {
			d.month = nums[0]
			d.day = nums[1]
		} else if hasMon {
			d.year = nums[1]
			if nums[1] < 100 {
				d.year += 1900
			}
			if nums[1] < 50 {
				d.year += 100
			}
			d.day = nums[0]
		} else {
			d.month = nums[0]
			d.day = nums[1]
		}
	case 3:
		d.year = nums[2]
		if nums[2] < 100 {
			d.year += 1900
		}
		if nums[2] < 50 {
			d.year += 100
		}
		d.month = nums[0]
		d.day = nums[1]
	default:
		return
	}
	if d.month < 1 || d.day < 1 {
		return
	}
	d.isLocal = d.isLocal && hasTime
	ok = true
	return
}
