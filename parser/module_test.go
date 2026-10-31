package parser

import "testing"

func TestParseModuleDeclarations(t *testing.T) {
	src := `import "side";
import d from "default";
import * as ns from "ns";
import d2, {a, b as c} from "mixed";
export default 1;
export {a, b as c};
export {x} from "reexport";
export * from "star";
export * as nsx from "star-ns";
export const q = 2;
`
	if _, err := ParseModule("test.js", src); err != nil {
		t.Fatal(err)
	}
}

func TestParseModuleRejectsInScript(t *testing.T) {
	if _, err := ParseFile(nil, "test.js", `import "x";`, 0); err == nil {
		t.Fatal("expected a syntax error for import in a script")
	}
}
