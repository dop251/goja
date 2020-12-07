package goja

import (
	"hash/maphash"
	"math"
	"strconv"
	"testing"
)

func testMapHashVal(v1, v2 Value, expected bool, t *testing.T) {
	var h maphash.Hash
	actual := v1.hash(&h) == v2.hash(&h)
	if actual != expected {
		t.Fatalf("testMapHashVal failed for %v, %v", v1, v2)
	}
}

func TestMapHash(t *testing.T) {
	testMapHashVal(_NaN, _NaN, true, t)
	testMapHashVal(valueTrue, valueFalse, false, t)
	testMapHashVal(valueTrue, valueTrue, true, t)
	testMapHashVal(intToValue(0), _negativeZero, true, t)
	testMapHashVal(asciiString("Test"), asciiString("Test"), true, t)
	testMapHashVal(newStringValue("Тест"), newStringValue("Тест"), true, t)
	testMapHashVal(floatToValue(1.2345), floatToValue(1.2345), true, t)
	testMapHashVal(SymIterator, SymToStringTag, false, t)
	testMapHashVal(SymIterator, SymIterator, true, t)

	// The following tests introduce indeterministic behaviour
	//testMapHashVal(asciiString("Test"), asciiString("Test1"), false, t)
	//testMapHashVal(newStringValue("Тест"), asciiString("Test"), false, t)
	//testMapHashVal(newStringValue("Тест"), newStringValue("Тест1"), false, t)
}

func TestOrderedMap(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
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

func TestOrderedMapCollision(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
	n1 := uint64(123456789)
	n2 := math.Float64frombits(n1)
	n1Key := intToValue(int64(n1))
	n2Key := floatToValue(n2)
	m.set(n1Key, asciiString("n1"))
	m.set(n2Key, asciiString("n2"))
	if m.size == len(m.hashTable) {
		t.Fatal("Expected a collision but there wasn't one")
	}
	if n2Val := m.get(n2Key); !asciiString("n2").SameAs(n2Val) {
		t.Fatalf("unexpected n2Val: %v", n2Val)
	}
	if n1Val := m.get(n1Key); !asciiString("n1").SameAs(n1Val) {
		t.Fatalf("unexpected nVal: %v", n1Val)
	}

	if !m.remove(n1Key) {
		t.Fatal("removing n1Key returned false")
	}
	if n2Val := m.get(n2Key); !asciiString("n2").SameAs(n2Val) {
		t.Fatalf("2: unexpected n2Val: %v", n2Val)
	}
}

func TestOrderedMapIter(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
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

func TestOrderedMapIterVisitAfterReAdd(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
	one := intToValue(1)
	two := intToValue(2)

	m.set(one, valueTrue)
	m.set(two, valueTrue)
	iter := m.newIter()
	entry := iter.next()
	if !one.SameAs(entry.key) {
		t.Fatalf("1: unexpected key: %v", entry.key)
	}
	if !m.remove(one) {
		t.Fatal("remove returned false")
	}
	entry = iter.next()
	if !two.SameAs(entry.key) {
		t.Fatalf("2: unexpected key: %v", entry.key)
	}
	m.set(one, valueTrue)
	entry = iter.next()
	if entry == nil {
		t.Fatal("entry is nil")
	}
	if !one.SameAs(entry.key) {
		t.Fatalf("3: unexpected key: %v", entry.key)
	}
}

func TestOrderedMapIterAddAfterClear(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
	one := intToValue(1)
	m.set(one, valueTrue)
	iter := m.newIter()
	iter.next()
	m.clear()
	m.set(one, valueTrue)
	entry := iter.next()
	if entry == nil {
		t.Fatal("entry is nil")
	}
	if entry.key != one {
		t.Fatalf("unexpected key: %v", entry.key)
	}
	entry = iter.next()
	if entry != nil {
		t.Fatalf("entry is not nil: %v", entry)
	}
}

func TestOrderedMapIterDeleteCurrent(t *testing.T) {
	m := newOrderedMap(&maphash.Hash{})
	one := intToValue(1)
	two := intToValue(2)
	iter := m.newIter()
	m.set(one, valueTrue)
	m.set(two, valueTrue)
	entry := iter.next()
	if entry.key != one {
		t.Fatalf("unexpected key: %v", entry.key)
	}
	m.remove(one)
	entry = iter.next()
	if entry.key != two {
		t.Fatalf("2: unexpected key: %v", entry.key)
	}
}
