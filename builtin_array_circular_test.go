package goja

import (
	"testing"
)

func TestArrayCircularReferenceToString(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	var str = String(T);
	// Circular reference should be replaced with empty string
	str === "1,2,3,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceNumericOperation(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	try {
		var x = T % 2;  // This should not crash
		true;
	} catch (e) {
		false;
	}
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceJoin(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	var str = T.join(',');
	// Circular reference should be replaced with empty string
	str === "1,2,3,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceConcat(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	var str = '' + T;  // String concatenation
	// Circular reference should be replaced with empty string
	str === "1,2,3,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceToLocaleString(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	var str = T.toLocaleString();
	// Circular reference should be replaced with empty string
	str === "1,2,3,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayMultipleCircularReferences(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;
	T[76] = T;
	T[80] = T;
	var str = String(T);
	// Should handle multiple circular references - all should be empty strings
	str.split(',').length === 81;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayNestedCircularReference(t *testing.T) {
	const SCRIPT = `
	var A = [1, 2];
	var B = [3, 4];
	A[2] = B;
	B[2] = A;  // Mutual circular reference
	var str = String(A);
	// A contains B which contains A - circular refs should be empty
	str === "1,2,3,4,";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceAccessOK(t *testing.T) {
	const SCRIPT = `
	// These operations should still work fine
	var T = [1, 2, 3];
	T[42] = T;
	
	// Accessing circular reference is OK
	var same = T[42] === T;
	
	// Accessing elements through circular reference is OK
	var first = T[42][0];
	
	// Deep nesting is OK
	var deep = T[42][42][42][42][42][0];
	
	same && first === 1 && deep === 1;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayCircularReferenceComparison(t *testing.T) {
	const SCRIPT = `
	var T = [1, 2, 3];
	T[42] = T;  // Create circular reference
	try {
		var result = T == 5;  // Comparison should not crash
		true;
	} catch (e) {
		false;
	}
	`
	testScript(SCRIPT, valueTrue, t)
}
