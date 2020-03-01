package goja

import (
	"strconv"
	"testing"
)

func testMapHashVal(v1, v2 Value, expected bool, t *testing.T) {
	actual := mapHash(v1) == mapHash(v2)
	if actual != expected {
		t.Fatalf("testMapHashVal failed for %v, %v", v1, v2)
	}
}

func TestMapHash(t *testing.T) {
	testMapHashVal(_NaN, _NaN, true, t)
	testMapHashVal(valueTrue, valueFalse, false, t)
	testMapHashVal(valueTrue, valueTrue, true, t)
	testMapHashVal(intToValue(0), _negativeZero, true, t)
	//testMapHashVal(asciiString("Test"), asciiString("Test1"), false, t)
	testMapHashVal(asciiString("Test"), asciiString("Test"), true, t)
	//testMapHashVal(newStringValue("Тест"), asciiString("Test"), false, t)
	testMapHashVal(newStringValue("Тест"), newStringValue("Тест"), true, t)
	//testMapHashVal(newStringValue("Тест"), newStringValue("Тест1"), false, t)
	testMapHashVal(floatToValue(1.2345), floatToValue(1.2345), true, t)
	testMapHashVal(symIterator, symToStringTag, false, t)
	testMapHashVal(symIterator, symIterator, true, t)
}

func TestOrderedMap(t *testing.T) {
	m := newOrderedMap()
	for i := int64(0); i < 50; i++ {
		m.set(intToValue(i), asciiString(strconv.FormatInt(i, 10)))
	}
	if m.size != 50 {
		t.Fatalf("Unexpected size: %d", m.size)
	}

	for i := int64(0); i < 50; i++ {
		expected := asciiString(strconv.FormatInt(i, 10))
		actual := m.get(intToValue(i))
		if !expected.SameAs(actual) {
			t.Fatalf("Wrong value for %d", i)
		}
	}

	for i := int64(0); i < 50; i += 2 {
		if !m.remove(intToValue(i)) {
			t.Fatalf("remove(%d) return false", i)
		}
	}
	if m.size != 25 {
		t.Fatalf("Unexpected size: %d", m.size)
	}

	iter := m.newIter()
	count := 0
	for {
		entry := iter.next()
		if entry == nil {
			break
		}
		m.remove(entry.key)
		count++
	}

	if count != 25 {
		t.Fatalf("Unexpected iter count: %d", count)
	}

	if m.size != 0 {
		t.Fatalf("Unexpected size: %d", m.size)
	}
}

func TestOrderedMapIter(t *testing.T) {
	m := newOrderedMap()
	iter := m.newIter()
	ent := iter.next()
	if ent != nil {
		t.Fatal("entry should be nil")
	}
	iter1 := m.newIter()
	m.set(intToValue(1), valueTrue)
	ent = iter.next()
	if ent != nil {
		t.Fatal("2: entry should be nil")
	}
	ent = iter1.next()
	if ent == nil {
		t.Fatal("entry is nil")
	}
	if !intToValue(1).SameAs(ent.key) {
		t.Fatal("unexpected key")
	}
	if !valueTrue.SameAs(ent.value) {
		t.Fatal("unexpected value")
	}
}
