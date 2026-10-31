# Intl

goja implements the ECMAScript Internationalization API on top of generated
CLDR 49 (Unicode 16) data. All data files are generated from
[cldr-json](https://github.com/unicode-org/cldr-json) and marked
`DO NOT EDIT`; regenerate them rather than editing by hand.

## Implemented

- `Intl` object: `getCanonicalLocales`, `supportedValuesOf`.
- `Intl.Locale`: full UTS 35 tag parser/canonicalizer (language, script,
  territory and variant aliases, territory replacement via likely subtags,
  `u`/`t` extension canonicalization, grandfathered tags), `maximize` /
  `minimize`, and `getCalendars`, `getCollations`, `getHourCycles`,
  `getNumberingSystems`, `getTextInfo`, `getTimeZones`, `getWeekInfo`.
- `Intl.PluralRules` (cardinal and ordinal, `select`, `selectRange`,
  rounding options).
- `Intl.NumberFormat` (decimal/percent/currency/unit styles, grouping,
  significant digits, `signDisplay`, accounting currency, scientific and
  engineering notation, non-Latin numbering systems, `formatToParts`).
- `Intl.DateTimeFormat` (dateStyle/timeStyle and component options, CLDR
  skeleton matching, time zones, `formatToParts`, `resolvedOptions`).
- `Intl.Collator` (sort/search, sensitivity, numeric, caseFirst,
  ignorePunctuation, collation resolution).
- `Intl.ListFormat`, `Intl.RelativeTimeFormat`, `Intl.DisplayNames`,
  `Intl.Segmenter`, `Intl.DurationFormat`.
- Locale-aware methods: `String.prototype.localeCompare`,
  `toLocaleLowerCase`/`toLocaleUpperCase`, and `toLocaleString` on
  `Number`, `BigInt`, `Date`, `Array` and typed arrays.

## Data coverage

Numbers, list patterns, relative-time fields, date patterns/names and
currency names cover 76 locales; display names cover 20 locales (English
fallback). Unknown locales fall back to `en`.

## Test status (test262 `intl402` via Mycel)

| Area | Passing |
| --- | --- |
| Locale | 150 / 152 |
| NumberFormat | 131 / 252 |
| DateTimeFormat | 123 / 247 |
| Segmenter | 72 / 77 |
| ListFormat | 67 / 80 |
| Collator | 60 / 64 |
| DisplayNames | 55 / 56 |
| RelativeTimeFormat | 54 / 79 |
| DurationFormat | 53 / 111 |
| PluralRules | 49 / 51 |
| Intl | 56 / 66 |
| Number/BigInt/Date/String/Array/TypedArray | 34 / 41 |

Before this implementation goja had no `Intl` at all (0 passing).

## Known gaps

- Compact notation is not implemented in `Intl.NumberFormat` (the CLDR
  compact patterns are not part of the generated data yet).
- `Intl.DateTimeFormat` does not yet model non-Gregorian calendars beyond
  resolving the option, and `formatRange` does not collapse shared parts.
- `Intl.RelativeTimeFormat` and `Intl.DurationFormat` still miss some
  style/display edge cases.
- `intl402/Temporal` requires the Temporal implementation and is not part
  of this work.
- Locale data is generated for a fixed set of locales; adding more is a
  matter of extending the generator inputs.
