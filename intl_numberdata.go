package goja

// intlNumberSymbols holds the CLDR number symbols for a locale.
type intlNumberSymbols struct {
	Decimal                string
	Group                  string
	List                   string
	PercentSign            string
	PlusSign               string
	MinusSign              string
	ApproximatelySign      string
	Exponential            string
	SuperscriptingExponent string
	PerMille               string
	Infinity               string
	NaN                    string
	TimeSeparator          string
}

// intlCurrencyInfo holds CLDR currency display data for a locale.
type intlCurrencyInfo struct {
	Symbol    string
	Narrow    string
	Name      string
	NameOne   string
	NameOther string
	Digits    int
}

// intlNumberLocale holds CLDR number formatting data for a locale.
type intlNumberLocale struct {
	DefaultNS          string
	MinGrouping        int
	Symbols            intlNumberSymbols
	Decimal            string
	Scientific         string
	Percent            string
	Currency           string
	CurrencyAccounting string
	Currencies         map[string]intlCurrencyInfo
}

func intlNumberLocaleFor(key string) *intlNumberLocale {
	if d, ok := intlNumberLocales[key]; ok {
		return d
	}
	return intlNumberLocales["en"]
}

// intlNumberSymbolsFor returns the symbols adjusted for the numbering system.
func intlNumberSymbolsFor(loc *intlNumberLocale, numberingSystem string) intlNumberSymbols {
	sym := loc.Symbols
	if numberingSystem != "" && numberingSystem != "latn" {
		if digits, ok := intlNumberingSystemDigits[numberingSystem]; ok {
			sym.Decimal = "."
			sym.Group = ","
			_ = digits
		}
	}
	return sym
}

func intlNumberingDigits(ns string) string {
	if d, ok := intlNumberingSystemDigits[ns]; ok {
		return d
	}
	return "0123456789"
}
