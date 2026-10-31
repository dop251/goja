package goja

import "testing"

func TestTemporalInstantBasics(t *testing.T) {
	const SCRIPT = `
	const inst = new Temporal.Instant(217178610123456789n);
	assert.sameValue(inst.epochNanoseconds, 217178610123456789n);
	assert.sameValue(inst.epochMilliseconds, 217178610123);
	assert.sameValue(Temporal.Instant.from("1976-11-18T15:23:30.123456789Z").epochNanoseconds, 217178610123456789n);
	assert.sameValue(Temporal.Instant.from("1976-11-18T16:23:30.123456789+01:00").epochNanoseconds, 217178610123456789n);
	assert.sameValue(Temporal.Instant.from("19761118T15:23:30.123456789Z").epochNanoseconds, 217178610123456789n);
	assert.sameValue(Temporal.Instant.fromEpochMilliseconds(217178610123).epochNanoseconds, 217178610123000000n);
	assert.sameValue(Temporal.Instant.fromEpochNanoseconds(5n).epochNanoseconds, 5n);
	assert.sameValue(new Temporal.Instant(-217175010876543211n).epochMilliseconds, -217175010877);
	assert.sameValue(inst.toString(), "1976-11-18T15:23:30.123456789Z");
	assert.sameValue(inst.toString({ smallestUnit: "second" }), "1976-11-18T15:23:30Z");
	assert.sameValue(inst.toString({ timeZone: "+01:00" }), "1976-11-18T16:23:30.123456789+01:00");
	assert.sameValue(inst.toString({ timeZone: "UTC" }), "1976-11-18T15:23:30.123456789+00:00");
	assert.throws(RangeError, () => new Temporal.Instant(8640000000000000000001n));
	assert.throws(TypeError, () => new Temporal.Instant(1));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalInstantArithmetic(t *testing.T) {
	const SCRIPT = `
	const i = Temporal.Instant.from("1970-01-01T00:00:00Z");
	assert.sameValue(i.add({ hours: 1 }).toString(), "1970-01-01T01:00:00Z");
	assert.sameValue(i.add({ nanoseconds: 1 }).toString(), "1970-01-01T00:00:00.000000001Z");
	assert.sameValue(i.subtract({ hours: 1 }).toString(), "1969-12-31T23:00:00Z");
	assert.throws(RangeError, () => i.add({ days: 1 }));
	assert.throws(RangeError, () => i.add({ months: 1 }));
	const d = i.until(Temporal.Instant.from("1970-01-01T02:30:00Z"), { largestUnit: "hours" });
	assert.sameValue(d.hours, 2);
	assert.sameValue(d.minutes, 30);
	const d2 = i.until(Temporal.Instant.from("1970-01-01T02:30:00Z"), { largestUnit: "minutes" });
	assert.sameValue(d2.minutes, 150);
	const d3 = i.since(Temporal.Instant.from("1970-01-01T02:30:00Z"), { largestUnit: "minutes" });
	assert.sameValue(d3.minutes, -150);
	assert.sameValue(i.equals(Temporal.Instant.from("1970-01-01T00:00:00Z")), true);
	assert.sameValue(Temporal.Instant.compare(i, Temporal.Instant.from("1970-01-01T00:00:01Z")), -1);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalInstantRounding(t *testing.T) {
	const SCRIPT = `
	const inst = Temporal.Instant.from("1976-11-18T14:23:30.123456789Z");
	assert.sameValue(inst.round({ smallestUnit: "hour" }).toString(), "1976-11-18T14:00:00Z");
	assert.sameValue(inst.round({ smallestUnit: "minute", roundingIncrement: 15 }).toString(), "1976-11-18T14:30:00Z");
	assert.sameValue(inst.round({ smallestUnit: "second", roundingIncrement: 30 }).toString(), "1976-11-18T14:23:30Z");
	assert.sameValue(inst.round({ smallestUnit: "hour", roundingIncrement: 24 }).toString(), "1976-11-19T00:00:00Z");
	const neg = new Temporal.Instant(-65261246399500000000n);
	assert.sameValue(neg.round({ smallestUnit: "second", roundingMode: "trunc" }).epochNanoseconds, -65261246400000000000n);
	assert.sameValue(neg.round({ smallestUnit: "second", roundingMode: "ceil" }).epochNanoseconds, -65261246399000000000n);
	assert.sameValue(Temporal.Instant.from("-000099-12-15T12:00:00.5Z").epochNanoseconds, -65261246399500000000n);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
