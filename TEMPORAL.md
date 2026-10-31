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
  go test -run 'TestTC39/tc39/test/built-ins/Temporal' -count=1
  ```

  `testdata/test262` may be a symlink to a local test262 checkout.

## Status

- [x] core (ISO arithmetic, float64-backed duration fields, exact `big.Rat` totals)
- [x] Duration: constructor, `from`, getters, `with`, `negated`, `abs`, `add`,
      `subtract`, `compare`, `total`, `round` (time/day units), `toString`,
      `toJSON`, `toLocaleString` — **378 / 533 tests pass**
- [ ] Duration relativeTo support (the remaining 155 tests need `PlainDate` /
      `PlainDateTime` / `ZonedDateTime`)
- [ ] PlainTime
- [ ] PlainDate
- [ ] PlainDateTime
- [ ] Instant
- [ ] PlainYearMonth / PlainMonthDay
- [ ] TimeZone / ZonedDateTime / Now

### Development placeholders

Until the remaining types are implemented, `Temporal` exposes placeholder
constructors for them. The placeholders throw when constructed or called; they
exist so test262 helper code that does `x instanceof Temporal.PlainDateTime`
keeps working while Duration is tested in isolation. They must be replaced by
the real implementations before this is submitted upstream.
