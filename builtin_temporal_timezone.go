package goja

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// temporalZone is a parsed time zone: either a fixed UTC offset or an IANA
// location backed by Go's time zone database.
type temporalZone struct {
	id     string
	offset *big.Int // non-nil for fixed offsets
	loc    *time.Location
}

func (z *temporalZone) fixed() bool { return z.offset != nil }

// offsetAt returns the UTC offset in nanoseconds in effect at an instant.
func (z *temporalZone) offsetAt(ns *big.Int) *big.Int {
	if z.offset != nil {
		return new(big.Int).Set(z.offset)
	}
	sec, _ := splitEpochSeconds(ns)
	_, off := time.Unix(sec, 0).In(z.loc).Zone()
	return big.NewInt(int64(off) * 1e9)
}

// offsetAtLocal returns the offset that applies to a local wall-clock time.
func (z *temporalZone) offsetAtLocal(localNs *big.Int) *big.Int {
	inst := z.instantFromLocal(localNs)
	if inst == nil {
		return big.NewInt(0)
	}
	return z.offsetAt(inst)
}

// localDateTime converts an instant to the wall-clock date and time in z.
func (z *temporalZone) localDateTime(ns *big.Int) (isoDate, isoTime, bool) {
	local := new(big.Int).Add(ns, z.offsetAt(ns))
	return isoDateTimeFromEpochNs(local)
}

// instantFromLocal converts a wall-clock time in z to an instant using the
// spec's "compatible" disambiguation.
func (z *temporalZone) instantFromLocal(localNs *big.Int) *big.Int {
	return z.instantFromLocalDisambiguation(localNs, "compatible")
}

// instantFromLocalDisambiguation resolves a wall-clock time to an instant.
// mode is one of "compatible", "earlier", "later" or "reject".
func (z *temporalZone) instantFromLocalDisambiguation(localNs *big.Int, mode string) *big.Int {
	if z.offset != nil {
		return new(big.Int).Sub(localNs, z.offset)
	}
	dayNs := temporalBigDayNs
	offBefore := z.offsetAt(new(big.Int).Sub(localNs, dayNs))
	offAfter := z.offsetAt(new(big.Int).Add(localNs, dayNs))
	candBefore := new(big.Int).Sub(localNs, offBefore)
	candAfter := new(big.Int).Sub(localNs, offAfter)
	validBefore := z.offsetAt(candBefore).Cmp(offBefore) == 0
	validAfter := z.offsetAt(candAfter).Cmp(offAfter) == 0
	switch {
	case validBefore && validAfter && offBefore.Cmp(offAfter) != 0:
		// Overlap: two possible instants.
		earlier, later := candBefore, candAfter
		if earlier.Cmp(later) > 0 {
			earlier, later = later, earlier
		}
		switch mode {
		case "earlier":
			return earlier
		case "later":
			return later
		case "reject":
			return nil
		default:
			return earlier
		}
	case validBefore && !validAfter:
		return candBefore
	case validAfter && !validBefore:
		return candAfter
	case validBefore && validAfter:
		return candBefore
	default:
		// Gap: the wall time does not exist.
		earlier, later := candBefore, candAfter
		if earlier.Cmp(later) > 0 {
			earlier, later = later, earlier
		}
		switch mode {
		case "earlier":
			return earlier
		case "reject":
			return nil
		default:
			return later
		}
	}
}

func splitEpochSeconds(ns *big.Int) (int64, int64) {
	sec, rem := new(big.Int).QuoRem(ns, big.NewInt(1e9), new(big.Int))
	if rem.Sign() < 0 {
		sec.Sub(sec, big.NewInt(1))
		rem.Add(rem, big.NewInt(1e9))
	}
	return sec.Int64(), rem.Int64()
}

// formatOffsetNano formats a UTC offset as ±HH:MM, adding seconds and
// fractional seconds only when they are non-zero.
func formatOffsetNano(off *big.Int) string {
	neg := off.Sign() < 0
	v := new(big.Int).Abs(off)
	sec := new(big.Int).Quo(v, big.NewInt(1e9)).Int64()
	frac := new(big.Int).Mod(v, big.NewInt(1e9)).Int64()
	sign := "+"
	if neg {
		sign = "-"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	out := fmt.Sprintf("%s%02d:%02d", sign, h, m)
	if s != 0 || frac != 0 {
		out += fmt.Sprintf(":%02d", s)
	}
	if frac != 0 {
		f := strings.TrimRight(fmt.Sprintf("%09d", frac), "0")
		out += "." + f
	}
	return out
}

// parseTemporalZone parses a time zone identifier: a UTC offset, a well-known
// UTC name, or an IANA location.
// parseTemporalZoneIdentifierStrict parses a bare time zone identifier, as
// required by the ZonedDateTime constructor (no ISO date-time strings).
func (r *Runtime) parseTemporalZoneIdentifierStrict(v Value) *temporalZone {
	s, ok := v.(String)
	if !ok {
		panic(r.NewTypeError("Time zone must be a string"))
	}
	str := s.String()
	if strings.Contains(str, "[") {
		panic(r.newError(r.getRangeError(), "Invalid time zone: "+str))
	}
	for i := 1; i < len(str); i++ {
		if (str[i] == 'T' || str[i] == 't') && str[i-1] >= '0' && str[i-1] <= '9' {
			panic(r.newError(r.getRangeError(), "Invalid time zone: "+str))
		}
	}
	return r.parseTemporalZoneString(str)
}

func (r *Runtime) parseTemporalZone(v Value) *temporalZone {
	if o, ok := v.(*Object); ok {
		if z, ok := o.self.(*temporalZonedDateTimeObject); ok {
			return z.zone
		}
	}
	s, ok := v.(String)
	if !ok {
		panic(r.NewTypeError("Time zone must be a string"))
	}
	str := s.String()
	return r.parseTemporalZoneString(str)
}

func (r *Runtime) parseTemporalZoneString(str string) *temporalZone {
	switch {
	case asciiEqualFold(str, "UTC"), asciiEqualFold(str, "Etc/UTC"):
		loc, _ := time.LoadLocation("UTC")
		return &temporalZone{id: "UTC", loc: loc}
	case asciiEqualFold(str, "GMT"):
		loc, _ := time.LoadLocation("GMT")
		return &temporalZone{id: "GMT", loc: loc}
	}
	if validTimeZoneOffsetLiteral(str) {
		off, _ := parseTemporalOffsetString(str)
		return &temporalZone{id: formatOffsetNano(off), offset: off}
	}
	if off, ok := r.parseTimeZoneOffsetString(str); ok {
		if timeZoneStringIsUTC(str) {
			loc, _ := time.LoadLocation("UTC")
			return &temporalZone{id: "UTC", loc: loc}
		}
		return &temporalZone{id: formatOffsetNano(off), offset: off}
	}
	if str == "" {
		panic(r.newError(r.getRangeError(), "Invalid time zone"))
	}
	if loc := loadLocationFold(str); loc != nil {
		return &temporalZone{id: loc.String(), loc: loc}
	}
	panic(r.newError(r.getRangeError(), "Invalid time zone: "+str))
}

// timeZoneStringIsUTC reports whether an ISO date-time time zone string is
// anchored to UTC (a Z designator or a UTC annotation).
func timeZoneStringIsUTC(str string) bool {
	switch str {
	case "Z", "z", "UTC", "Etc/UTC", "GMT":
		return true
	}
	base, anns, ok := splitTemporalAnnotations(str)
	if !ok {
		return false
	}
	for _, ann := range anns {
		ann = strings.TrimPrefix(ann, "!")
		if strings.Contains(ann, "=") {
			continue
		}
		if isTimeZoneAnnotation(ann) {
			return ann == "UTC" || ann == "Etc/UTC"
		}
	}
	return strings.HasSuffix(base, "Z") || strings.HasSuffix(base, "z")
}

// loadLocationFold looks up an IANA name case-insensitively, returning the
// canonical location.
func loadLocationFold(name string) *time.Location {
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	// Try a few common casings.
	candidates := []string{strings.ToLower(name), strings.ToUpper(name)}
	parts := strings.Split(name, "/")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	candidates = append(candidates, strings.Join(parts, "/"))
	for _, c := range candidates {
		if loc, err := time.LoadLocation(c); err == nil {
			if strings.EqualFold(loc.String(), name) {
				return loc
			}
		}
	}
	return nil
}

var _ = time.UTC
