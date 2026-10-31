# Temporal implementation plan

This branch is a staging ground for adding [Temporal](https://tc39.es/proposal-temporal/)
to goja. It is structured so the work can be submitted upstream as a series of
reviewable PRs instead of one unreviewable change.

## Scope

- **Initial scope:** ISO 8601 calendar, proleptic Gregorian date math, IANA time
  zones via Go's `time` package, `Temporal.Now` backed by `time.Now`.
- **Out of scope initially:** non-ISO calendars (`japanese`, `hebrew`, `chinese`,
  ...) and locale formatting. Those depend on CLDR/Intl data that goja does not
  have. ECMA-402 Temporal tests remain skipped.

## Milestones

| PR | Content | Test262 focus |
| --- | --- | --- |
| 1 | internal temporal core: epoch seconds+nanos, civil date algorithms, duration record, rounding | — |
| 2 | `Temporal.Duration` | `built-ins/Temporal/Duration` (533) |
| 3 | `Temporal.PlainTime`, `Temporal.PlainDate` | 493 + 648 |
| 4 | `Temporal.PlainDateTime`, `Temporal.Instant` | 769 + 465 |
| 5 | `Temporal.PlainYearMonth`, `Temporal.PlainMonthDay` | 706 |
| 6 | `Temporal.TimeZone`, `Temporal.ZonedDateTime`, `Temporal.Now` | 965 |
| 7 | optional non-ISO calendars / Intl integration | `intl402/Temporal` (2006) |

Each milestone removes the corresponding tests from
`test/built-ins/Temporal` and removes `Temporal` from the feature blacklist
once the namespace is complete.

## Implementation notes

- **Instants** are `(seconds int64, nanoseconds int32)` rather than a single
  nanosecond integer: the Temporal instant range (±8.64e21 ns) does not fit in
  int64.
- **Durations** store the ten spec fields as int64. Range validation uses
  `math/big` because the total number of nanoseconds can exceed int64.
- **Time zones** use `time.LoadLocation` / `time.Time.Zone()`; tests are run with
  the system zoneinfo database (optionally `time/tzdata`).
- **Brand checks** use Go type assertions on the internal object structs, the
  same pattern as `Map`/`Set`.
- **Property layout** follows the built-in requirements: getters on prototypes,
  correct `length`/`name`, non-enumerable methods, `Symbol.toStringTag`.
- Tests are run with the upstream `tc39_test.go` runner:

  ```sh
  GOJA_TEMPORAL_TESTS=1 go test -run 'TestTC39/tc39/test/built-ins/Temporal/Duration$/' -count=1
  GOJA_TEMPORAL_TESTS=1 go test -run 'TestTC39/tc39/test/built-ins/Temporal/PlainDate$/' -count=1
  ```

  The default test run keeps `Temporal` in the feature blacklist until all
  types are implemented, so ordinary CI stays green.

  `testdata/test262` may be a symlink to a local test262 checkout.

## Status

- [x] core (ISO arithmetic, exact `big.Rat`/`big.Int` totals)
- [x] Duration: constructor, `from`, getters, `with`, `negated`, `abs`, `add`,
      `subtract`, `compare`, `total`, `round`, `toString`, `toJSON`,
      `toLocaleString`, including `relativeTo` for `PlainDate`,
      `PlainDateTime` and `ZonedDateTime` — **470 / 533 tests pass**
- [x] PlainDate: constructor, `from`, getters, `with`, `withCalendar`, `add`,
      `subtract`, `until`, `since`, `equals`, `compare`, `toString`,
      `toJSON`, `toPlainDateTime`, `toZonedDateTime`, `toPlainYearMonth`,
      `toPlainMonthDay`, `getISOFields` — **648 / 648 tests pass**
- [x] PlainTime: constructor, `from`, getters, `with`, `add`, `subtract`,
      `until`, `since`, `round`, `equals`, `compare`, `toString`, `toJSON`,
      `getISOFields`, `toPlainDateTime` — **493 / 493 tests pass**
- [x] PlainDateTime: constructor, `from`, getters, `with`, `withCalendar`,
      `withPlainTime`, `add`, `subtract`, `until`, `since`, `round`, `equals`,
      `compare`, `toString`, `toJSON`, `toPlainDate`, `toPlainTime`,
      `toZonedDateTime`, `toPlainYearMonth`, `toPlainMonthDay`,
      `getISOFields` — **769 / 769 tests pass**
- [x] Instant: constructor, `from`, `fromEpochMilliseconds`,
      `fromEpochNanoseconds`, `epochMilliseconds`, `epochNanoseconds`, `add`,
      `subtract`, `until`, `since`, `round`, `equals`, `compare`, `toString`,
      `toJSON`, `toLocaleString`, `toZonedDateTimeISO` — **465 / 465 tests
      pass**
- [x] ZonedDateTime: IANA and fixed-offset time zones, DST-aware wall-clock
      arithmetic, `getTimeZoneTransition`, and the full date/time surface —
      **894 / 899 tests pass** (remaining failures are a few ISO-limit edge
      cases)
- [x] PlainYearMonth: constructor, `from`, getters, `with`, `add`, `subtract`,
      `until`, `since`, `equals`, `compare`, `toString`, `toJSON`,
      `toPlainDate`, `getISOFields` — **507 / 507 tests pass**
- [x] PlainMonthDay: constructor, `from`, getters, `with`, `equals`,
      `toString`, `toJSON`, `toPlainDate`, `getISOFields` — **199 / 199 tests
      pass**
- [x] `Temporal.Now` (system clock, optional time zone)
- [ ] exact calendar-unit `Duration.total`/`compare` for a few remaining
      relative-to cases
- [ ] non-ISO calendars / Intl integration

The remaining Duration failures are concentrated in `total`/`compare` for
durations that mix calendar and time units, plus `relativeTo` strings that
carry time zone offsets.
