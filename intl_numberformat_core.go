package goja

import (
	"math"
	"strconv"
	"strings"
)

// intlNumberFormatOptions holds the resolved number formatting options.
type intlNumberFormatOptions struct {
	style               string // decimal, percent, currency, unit
	currency            string
	currencyDisplay     string // code, symbol, narrowSymbol, name
	currencySign        string // standard, accounting
	unit                string
	unitDisplay         string // long, short, narrow
	notation            string // standard, scientific, engineering, compact
	compactDisplay      string
	useGrouping         string // auto, always, never, min2, min5
	signDisplay         string
	minInt              int
	minFrac, maxFrac    int
	minSig, maxSig      int // 0 == unused
	roundingIncrement   int
	roundingMode        string
	roundingPriority    string
	trailingZeroDisplay string
	numberingSystem     string
	fractionSpecified   bool
}

func intlDefaultNumberOptions() *intlNumberFormatOptions {
	return &intlNumberFormatOptions{
		style:               "decimal",
		currencyDisplay:     "symbol",
		currencySign:        "standard",
		unitDisplay:         "short",
		notation:            "standard",
		compactDisplay:      "short",
		useGrouping:         "auto",
		signDisplay:         "auto",
		minInt:              1,
		minFrac:             0,
		maxFrac:             3,
		roundingIncrement:   1,
		roundingMode:        "halfExpand",
		roundingPriority:    "auto",
		trailingZeroDisplay: "auto",
	}
}

// intlFormatNumberParts formats n and returns the parts.
func intlFormatNumberParts(n float64, opts *intlNumberFormatOptions, loc *intlNumberLocale) []intlStringPart {
	sym := loc.Symbols
	ns := opts.numberingSystem
	if ns == "" {
		ns = loc.DefaultNS
	}
	digits := intlNumberingDigits(ns)

	if math.IsNaN(n) {
		return []intlStringPart{{"nan", sym.NaN}}
	}
	if math.IsInf(n, 0) {
		var parts []intlStringPart
		if n < 0 {
			parts = append(parts, intlStringPart{"minusSign", sym.MinusSign})
		} else if opts.signDisplay == "always" || opts.signDisplay == "exceptZero" {
			parts = append(parts, intlStringPart{"plusSign", sym.PlusSign})
		}
		return append(parts, intlStringPart{"infinity", sym.Infinity})
	}

	negative := intlNumberIsNegative(n)
	if negative {
		n = -n
	}
	if opts.notation == "scientific" || opts.notation == "engineering" ||
		(opts.notation == "standard" && n != 0 && (n >= 1e21 || n < 1e-6)) {
		return intlFormatExponentialParts(n, negative, opts, loc, sym)
	}
	scale := 1.0
	switch opts.style {
	case "percent":
		scale = 100
	case "unit":
		if opts.unit == "percent" {
			scale = 100
		}
	}
	n *= scale

	minFrac, maxFrac := opts.minFrac, opts.maxFrac
	if opts.style == "currency" {
		digitsCur := intlCurrencyDefaultDigitsValue
		if ci, ok := loc.Currencies[opts.currency]; ok {
			digitsCur = ci.Digits
		} else if d, ok := intlCurrencyDefaultDigits[opts.currency]; ok {
			digitsCur = d
		}
		if !opts.fractionSpecified {
			minFrac, maxFrac = digitsCur, digitsCur
		}
	}
	var intPart, fracPart string
	if opts.minSig > 0 || opts.maxSig > 0 {
		_, intPart, fracPart = intlDecimalDigits(n)
		intPart, fracPart = intlRoundToSignificant(intPart, fracPart, opts.minSig, opts.maxSig, opts.roundingMode)
	} else {
		_, intPart, fracPart = intlDecimalDigits(n)
		intPart, fracPart = intlRoundDigits(intPart, fracPart, maxFrac, opts.roundingMode)
	}
	// trailing zero display
	if opts.trailingZeroDisplay == "stripIfInteger" {
		if !intlHasNonZero(fracPart) {
			fracPart = ""
		}
	}
	for len(fracPart) > minFrac && strings.HasSuffix(fracPart, "0") {
		fracPart = fracPart[:len(fracPart)-1]
	}
	for len(fracPart) < minFrac {
		fracPart += "0"
	}
	for len(intPart) < opts.minInt {
		intPart = "0" + intPart
	}
	if intPart == "" {
		intPart = "0"
	}

	if ns != "" && ns != "latn" {
		if sym2, ok := intlNumberingSystemSymbols[ns]; ok {
			if sym2.Decimal != "" {
				sym.Decimal = sym2.Decimal
			}
			if sym2.Group != "" {
				sym.Group = sym2.Group
			}
		}
	}
	grouping := intlShouldGroup(opts, loc, len(intPart))
	grouped := intPart
	if grouping {
		grouped = intlGroupDigits(intPart, sym.Group, 3, 3)
	}

	// Determine affixes from the pattern.
	prefix, suffix := intlNumberPatternAffixes(opts, loc, sym, negative)

	var parts []intlStringPart
	if negative {
		accounting := opts.style == "currency" && opts.currencySign == "accounting" &&
			strings.Contains(loc.CurrencyAccounting, ";")
		switch opts.signDisplay {
		case "never", "negative", "negativeExceptZero":
			if !accounting && (opts.signDisplay == "negative" || opts.signDisplay == "negativeExceptZero") {
				parts = append(parts, intlStringPart{"minusSign", sym.MinusSign})
			}
		default:
			if !accounting {
				parts = append(parts, intlStringPart{"minusSign", sym.MinusSign})
			}
		}
	} else {
		if opts.signDisplay == "always" || opts.signDisplay == "exceptZero" {
			parts = append(parts, intlStringPart{"plusSign", sym.PlusSign})
		} else if opts.signDisplay == "negative" || opts.signDisplay == "negativeExceptZero" {
			parts = append(parts, intlStringPart{"plusSign", sym.PlusSign})
		}
	}

	appendString := func(s string) {
		if s == "" {
			return
		}
		if isASCII(s) {
			parts = append(parts, intlStringPart{"literal", s})
		} else {
			parts = append(parts, intlStringPart{"literal", s})
		}
	}

	appendPrefix := func(p string) {
		// Split the prefix around the currency/unit placeholder we already
		// resolved into plain text.
		appendString(p)
	}
	appendPrefix(prefix)
	parts = appendIntlDigits(parts, grouped, "integer", digits)
	if fracPart != "" {
		parts = append(parts, intlStringPart{"decimal", sym.Decimal})
		parts = appendIntlDigits(parts, fracPart, "fraction", digits)
	}
	appendString(suffix)
	return parts
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// appendIntlDigits appends grouped digit parts, adding "group" parts.
func appendIntlDigits(parts []intlStringPart, s, typ, digits string) []intlStringPart {
	// Split on the group separator by re-scanning: the caller has already
	// inserted group separators, so split them out here.
	if typ != "integer" {
		return append(parts, intlStringPart{typ, intlMapDigits(s, digits)})
	}
	segs := strings.Split(s, ",")
	for i, seg := range segs {
		if i > 0 {
			parts = append(parts, intlStringPart{"group", ","})
		}
		parts = append(parts, intlStringPart{typ, intlMapDigits(seg, digits)})
	}
	return parts
}

func intlMapDigits(s, digits string) string {
	if digits == "0123456789" {
		return s
	}
	runes := []rune(digits)
	var sb strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			sb.WriteRune(runes[c-'0'])
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func intlShouldGroup(opts *intlNumberFormatOptions, loc *intlNumberLocale, intLen int) bool {
	switch opts.useGrouping {
	case "never":
		return false
	case "always":
		return intLen > 3
	case "min2":
		return intLen > 4
	case "min5":
		return intLen > 7
	}
	minDigits := 3 + (loc.MinGrouping - 1)
	if minDigits < 3 {
		minDigits = 3
	}
	return intLen > minDigits
}

// intlNumberPatternAffixes computes the text before and after the number.
func intlNumberPatternAffixes(opts *intlNumberFormatOptions, loc *intlNumberLocale, sym intlNumberSymbols, negative bool) (string, string) {
	accounting := false
	pattern := loc.Decimal
	switch opts.style {
	case "percent":
		pattern = loc.Percent
	case "currency":
		pattern = loc.Currency
		if opts.currencySign == "accounting" {
			pattern = loc.CurrencyAccounting
		}
	case "unit":
		return "", " " + opts.unit
	}
	if i := strings.IndexByte(pattern, ';'); i >= 0 {
		if negative && opts.currencySign == "accounting" {
			pattern = pattern[i+1:]
			accounting = true
		} else {
			pattern = pattern[:i]
		}
	}
	_ = accounting
	// Strip the numeric portion to obtain the affixes.
	start := strings.IndexAny(pattern, "#0")
	if start < 0 {
		return "", ""
	}
	end := start
	for end < len(pattern) && (pattern[end] == '#' || pattern[end] == '0' || pattern[end] == ',' || pattern[end] == '.') {
		end++
	}
	prefix := pattern[:start]
	suffix := pattern[end:]

	currencyText := ""
	if opts.style == "currency" {
		ci := loc.Currencies[opts.currency]
		switch opts.currencyDisplay {
		case "code":
			currencyText = opts.currency
		case "name":
			currencyText = ci.Name
			if currencyText == "" {
				currencyText = opts.currency
			}
			return "", " " + currencyText
		case "narrowSymbol":
			currencyText = ci.Narrow
			if currencyText == "" {
				currencyText = ci.Symbol
			}
		default:
			currencyText = ci.Symbol
		}
		if currencyText == "" {
			currencyText = opts.currency
		}
	} else if opts.style == "percent" {
		currencyText = sym.PercentSign
	}
	prefix = strings.ReplaceAll(prefix, "¤", currencyText)
	suffix = strings.ReplaceAll(suffix, "¤", currencyText)
	suffix = strings.ReplaceAll(suffix, "%", sym.PercentSign)
	prefix = strings.ReplaceAll(prefix, "%", sym.PercentSign)
	return prefix, suffix
}

// intlFormatNumberString formats a number to a plain string (used by
// RelativeTimeFormat).
func intlFormatNumberString(n float64, loc *intlNumberLocale, opts *intlNumberFormatOptions) string {
	var sb strings.Builder
	for _, p := range intlFormatNumberParts(n, opts, loc) {
		sb.WriteString(p.value)
	}
	return sb.String()
}

var intlNumberingSystemSymbols = map[string]intlNumberSymbols{
	"arab":    {Decimal: "\u066b", Group: "\u066c"},
	"arabext": {Decimal: "\u066b", Group: "\u066c"},
}

// intlFormatExponentialParts formats n in scientific or engineering notation.
func intlFormatExponentialParts(n float64, negative bool, opts *intlNumberFormatOptions, loc *intlNumberLocale, sym intlNumberSymbols) []intlStringPart {
	ns := opts.numberingSystem
	if ns == "" {
		ns = loc.DefaultNS
	}
	digits := intlNumberingDigits(ns)
	var parts []intlStringPart
	if negative {
		switch opts.signDisplay {
		case "never":
		default:
			parts = append(parts, intlStringPart{"minusSign", sym.MinusSign})
		}
	} else if opts.signDisplay == "always" || opts.signDisplay == "exceptZero" {
		parts = append(parts, intlStringPart{"plusSign", sym.PlusSign})
	}
	if n == 0 {
		parts = appendIntlDigits(parts, "0", "integer", digits)
		return parts
	}
	exp := int(math.Floor(math.Log10(n)))
	if opts.notation == "engineering" {
		exp = int(math.Floor(float64(exp)/3.0)) * 3
	}
	mant := n / math.Pow(10, float64(exp))
	_, ip, fp := intlDecimalDigits(mant)
	ip, fp = intlRoundDigits(ip, fp, opts.maxFrac, opts.roundingMode)
	for len(fp) > opts.minFrac && strings.HasSuffix(fp, "0") {
		fp = fp[:len(fp)-1]
	}
	for len(fp) < opts.minFrac {
		fp += "0"
	}
	if ip == "" {
		ip = "0"
	}
	// Re-normalize if rounding pushed the mantissa over the notation bound.
	limit := 10.0
	if opts.notation == "engineering" {
		limit = 1000.0
	}
	if m := intlParseMantissa(ip, fp); m >= limit {
		if opts.notation == "engineering" {
			exp += 3
			m /= 1000
		} else {
			exp++
			m /= 10
		}
		ip2, fp2 := strconv.FormatFloat(m, 'f', -1, 64), ""
		if i := strings.IndexByte(ip2, '.'); i >= 0 {
			ip2, fp2 = ip2[:i], ip2[i+1:]
		}
		for len(fp2) > opts.minFrac && strings.HasSuffix(fp2, "0") {
			fp2 = fp2[:len(fp2)-1]
		}
		ip, fp = ip2, fp2
	}
	parts = appendIntlDigits(parts, ip, "integer", digits)
	if fp != "" {
		parts = append(parts, intlStringPart{"decimal", sym.Decimal})
		parts = appendIntlDigits(parts, fp, "fraction", digits)
	}
	parts = append(parts, intlStringPart{"exponentSeparator", sym.Exponential})
	if exp < 0 {
		parts = append(parts, intlStringPart{"exponentMinusSign", sym.MinusSign})
		exp = -exp
	}
	parts = appendIntlDigits(parts, strconv.Itoa(exp), "exponentInteger", digits)
	return parts
}

func intlParseMantissa(ip, fp string) float64 {
	v, _ := strconv.ParseFloat(ip+"."+fp, 64)
	return v
}
