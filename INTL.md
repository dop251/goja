# Intl implementation status

goja now ships an `Intl` namespace backed by generated CLDR 49 data
(`intl_cldr_*.go`, Unicode 16.0).

## Implemented

- `Intl` object, `Intl.getCanonicalLocales`, `Intl.supportedValuesOf`
- `Intl.Locale` with the full UTS 35 canonicalisation pipeline (language
  aliases, territory aliases that depend on likely subtags, variant aliases,
  `u`/`t` extension canonicalisation, grandfathered tags), `maximize` /
  `minimize`, and the `getCalendars`, `getCollations`, `getHourCycles`,
  `getNumberingSystems`, `getTextInfo`, `getTimeZones`, `getWeekInfo` APIs
- `Intl.PluralRules` (cardinal and ordinal, `select`, `selectRange`,
  significant digits, rounding options)
- `Intl.ListFormat` (conjunction/disjunction/unit, long/short/narrow,
  iterables, `formatToParts`)
- `Intl.RelativeTimeFormat` (long/short/narrow, `numeric: auto`,
  `formatToParts`, per-locale plural selection)
- `Intl.Collator` (sort/search, sensitivity, numeric, caseFirst,
  ignorePunctuation, collation resolution) and `String.prototype.localeCompare`
- locale-aware `toLocaleLowerCase`/`toLocaleUpperCase`
- `Intl.DateTimeFormat` (work in progress: dateStyle/timeStyle, component
  options, skeletons from CLDR availableFormats, time zones, formatToParts)
- `Intl.NumberFormat` (decimal/percent/currency/unit styles, grouping,
  significant digits, `signDisplay`, `currencySign: accounting`,
  scientific/engineering notation, `formatToParts`, numbering systems,
  currency symbols/names from CLDR)

## Not yet implemented

`Intl.Segmenter`,
`Intl.DisplayNames`, `Intl.DurationFormat`, and the locale methods on
`String`, `Number`, `BigInt`, `Date` and `Array` prototypes.
`intl402/Temporal` additionally requires the Temporal implementation.

## Data

`intl_cldr_locales.go` carries the number, list and relative-time data for
the 76 supported locales; other locales fall back to `en`.

## Tests

Mycel's Test262 run for `intl402/`: **~730 passing** (from 0 before this
work), including 150/152 `intl402/Locale`, 126 `intl402/NumberFormat`,
119/248 `intl402/DateTimeFormat`, 67 `intl402/ListFormat`,
59/64 `intl402/Collator`, 54 `intl402/RelativeTimeFormat` and 48
`intl402/PluralRules`.
