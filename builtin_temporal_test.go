package goja

import "testing"

func TestTemporalDurationBasics(t *testing.T) {
	const SCRIPT = `
	const d = new Temporal.Duration(1, 2, 3, 4, 5, 6, 7, 8, 9, 10);
	assert.sameValue(d.years, 1);
	assert.sameValue(d.months, 2);
	assert.sameValue(d.weeks, 3);
	assert.sameValue(d.days, 4);
	assert.sameValue(d.hours, 5);
	assert.sameValue(d.minutes, 6);
	assert.sameValue(d.seconds, 7);
	assert.sameValue(d.milliseconds, 8);
	assert.sameValue(d.microseconds, 9);
	assert.sameValue(d.nanoseconds, 10);
	assert.sameValue(d.sign, 1);
	assert.sameValue(d.blank, false);
	assert.sameValue(d.toString(), "P1Y2M3W4DT5H6M7.00800901S");

	const blank = new Temporal.Duration();
	assert.sameValue(blank.sign, 0);
	assert.sameValue(blank.blank, true);
	assert.sameValue(blank.toString(), "PT0S");
	assert.sameValue(Object.getPrototypeOf(Temporal.Duration.from(blank)), Temporal.Duration.prototype);
	assert.sameValue(Temporal.Duration.from(blank) === blank, false);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalDurationFrom(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(Temporal.Duration.from("P1Y2M3W4DT5H6M7.00800901S").toString(), "P1Y2M3W4DT5H6M7.00800901S");
	assert.sameValue(Temporal.Duration.from("p1y2m3w4dt5h6m7.00800901s").toString(), "P1Y2M3W4DT5H6M7.00800901S");
	assert.sameValue(Temporal.Duration.from("PT1.03125H").toString(), "PT1H1M52.5S");
	assert.sameValue(Temporal.Duration.from("PT3,025M").toString(), "PT3M1.5S");
	assert.sameValue(Temporal.Duration.from({ hours: 25 }).toString(), "PT25H");
	assert.throws(TypeError, () => Temporal.Duration.from({ month: 1 }));
	assert.throws(TypeError, () => Temporal.Duration.from({}));
	assert.throws(RangeError, () => Temporal.Duration.from({ hours: 0.5 }));
	assert.throws(RangeError, () => Temporal.Duration.from("P1Y1M1W1DT1H1M1.1234567891S"));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalDurationArithmetic(t *testing.T) {
	const SCRIPT = `
	const d = Temporal.Duration.from({ microseconds: Number.MAX_SAFE_INTEGER });
	assert.sameValue(d.add({ microseconds: Number.MAX_SAFE_INTEGER - 1 }).microseconds, 18014398509481980);
	assert.sameValue(Temporal.Duration.from({ hours: 25 }).round({ largestUnit: "days" }).toString(), "P1DT1H");
	assert.sameValue(Temporal.Duration.from({ seconds: 59, milliseconds: 900 }).toString({ fractionalSecondDigits: 0, roundingMode: "expand" }), "PT60S");
	assert.sameValue(Temporal.Duration.from({ hours: 1, minutes: 59, seconds: 59, milliseconds: 900 }).toString({ fractionalSecondDigits: 0, roundingMode: "expand" }), "PT2H0S");
	assert.sameValue(Temporal.Duration.from({ seconds: 120, milliseconds: 3500 }).toString(), "PT123.5S");
	assert.sameValue(Temporal.Duration.from({ milliseconds: 3500 }).toString(), "PT3.5S");
	assert.sameValue(Temporal.Duration.from({ days: 1, seconds: 1 }).total("seconds"), 86401);
	assert.sameValue(Temporal.Duration.compare({ hours: 1 }, { minutes: 60 }), 0);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
