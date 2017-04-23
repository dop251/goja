package goja

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

const (
	tc39BASE = "testdata/test262"
)

var (
	invalidFormatError = errors.New("Invalid file format")
)

var (
	skipList = map[string]bool{
		"test/language/literals/regexp/S7.8.5_A1.1_T2.js":             true, // UTF-16
		"test/language/literals/regexp/S7.8.5_A1.4_T2.js":             true, // UTF-16
		"test/language/literals/regexp/S7.8.5_A2.1_T2.js":             true, // UTF-16
		"test/language/literals/regexp/S7.8.5_A2.4_T2.js":             true, // UTF-16
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-9.js":  true, // timezone
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-10.js": true, // timezone
		"test/built-ins/Object/getOwnPropertyNames/15.2.3.4-4-44.js":  true, // property order
	}
)

type tc39TestCtx struct {
	prgCache map[string]*Program
}

type TC39MetaNegative struct {
	Phase, Type string
}

type tc39Meta struct {
	Negative TC39MetaNegative
	Includes []string
	Flags    []string
	Es5id    string
	Es6id    string
	Esid     string
}

func (m *tc39Meta) hasFlag(flag string) bool {
	for _, f := range m.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

func parseTC39File(name string) (*tc39Meta, string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	str := string(b)
	metaStart := strings.Index(str, "/*---")
	if metaStart == -1 {
		return nil, "", invalidFormatError
	} else {
		metaStart += 5
	}
	metaEnd := strings.Index(str, "---*/")
	if metaEnd == -1 || metaEnd <= metaStart {
		return nil, "", invalidFormatError
	}

	var meta tc39Meta
	err = yaml.Unmarshal([]byte(str[metaStart:metaEnd]), &meta)
	if err != nil {
		return nil, "", err
	}

	if meta.Negative.Type != "" && meta.Negative.Phase == "" {
		return nil, "", errors.New("negative type is set, but phase isn't")
	}

	return &meta, str, nil
}

func runTC39Test(base, name, src string, meta *tc39Meta, t testing.TB, ctx *tc39TestCtx) {
	vm := New()
	err, early := runTC39Script(base, name, src, meta.Includes, t, ctx, vm)

	if err != nil {
		if meta.Negative.Type == "" {
			t.Fatalf("%s: %v", name, err)
		} else {
			if meta.Negative.Phase == "early" && !early || meta.Negative.Phase == "runtime" && early {
				t.Fatalf("%s: error %v happened at the wrong phase (expected %s)", name, err, meta.Negative.Phase)
			}
			var errType string

			switch err := err.(type) {
			case *Exception:
				if o, ok := err.Value().(*Object); ok {
					if c := o.Get("constructor"); c != nil {
						if c, ok := c.(*Object); ok {
							errType = c.Get("name").String()
						} else {
							t.Fatalf("%s: error constructor is not an object (%v)", name, o)
						}
					} else {
						t.Fatalf("%s: error does not have a constructor (%v)", name, o)
					}
				} else {
					t.Fatalf("%s: error is not an object (%v)", name, err.Value())
				}
			case *CompilerSyntaxError:
				errType = "SyntaxError"
			case *CompilerReferenceError:
				errType = "ReferenceError"
			default:
				t.Fatalf("%s: error is not a JS error: %v", name, err)
			}

			if errType != meta.Negative.Type {
				vm.vm.prg.dumpCode(t.Logf)
				t.Fatalf("%s: unexpected error type (%s), expected (%s)", name, errType, meta.Negative.Type)
			}
		}
	} else {
		if meta.Negative.Type != "" {
			vm.vm.prg.dumpCode(t.Logf)
			t.Fatalf("%s: Expected error: %v", name, err)
		}
	}
}

func runTC39File(base, name string, t testing.TB, ctx *tc39TestCtx) {
	if skipList[name] {
		t.Logf("Skipped %s", name)
		return
	}
	p := path.Join(base, name)
	meta, src, err := parseTC39File(p)
	if err != nil {
		//t.Fatalf("Could not parse %s: %v", name, err)
		t.Errorf("Could not parse %s: %v", name, err)
		return
	}
	if meta.Es5id == "" {
		//t.Logf("%s: Not ES5, skipped", name)
		return
	}

	hasRaw := meta.hasFlag("raw")

	if hasRaw || !meta.hasFlag("onlyStrict") {
		//log.Printf("Running normal test: %s", name)
		//t.Logf("Running normal test: %s", name)
		runTC39Test(base, name, src, meta, t, ctx)
	}

	if !hasRaw && !meta.hasFlag("noStrict") {
		//log.Printf("Running strict test: %s", name)
		//t.Logf("Running strict test: %s", name)
		runTC39Test(base, name, "'use strict';\n"+src, meta, t, ctx)
	}

}

func (ctx *tc39TestCtx) runFile(base, name string, vm *Runtime) error {
	prg := ctx.prgCache[name]
	if prg == nil {
		fname := path.Join(base, name)
		f, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer f.Close()

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		str := string(b)
		prg, err = Compile(name, str, false)
		if err != nil {
			return err
		}
		ctx.prgCache[name] = prg
	}
	_, err := vm.RunProgram(prg)
	return err
}

func runTC39Script(base, name, src string, includes []string, t testing.TB, ctx *tc39TestCtx, vm *Runtime) (err error, early bool) {
	early = true
	err = ctx.runFile(base, path.Join("harness", "assert.js"), vm)
	if err != nil {
		return
	}

	err = ctx.runFile(base, path.Join("harness", "sta.js"), vm)
	if err != nil {
		return
	}

	for _, include := range includes {
		err = ctx.runFile(base, path.Join("harness", include), vm)
		if err != nil {
			return
		}
	}

	var p *Program
	p, err = Compile(name, src, false)

	if err != nil {
		return
	}

	early = false
	_, err = vm.RunProgram(p)

	return
}

func runTC39Tests(base, name string, t *testing.T, ctx *tc39TestCtx) {
	files, err := ioutil.ReadDir(path.Join(base, name))
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.Name()[0] == '.' {
			continue
		}
		if file.IsDir() {
			runTC39Tests(base, path.Join(name, file.Name()), t, ctx)
		} else {
			if strings.HasSuffix(file.Name(), ".js") {
				runTC39File(base, path.Join(name, file.Name()), t, ctx)
			}
		}
	}

}

func TestTC39(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, err := os.Stat(tc39BASE); err != nil {
		t.Skipf("If you want to run tc39 tests, download them from https://github.com/tc39/test262 and put into %s. The last working commit is 1ba3a7c4a93fc93b3d0d7e4146f59934a896837d. (%v)", tc39BASE, err)
	}

	ctx := &tc39TestCtx{
		prgCache: make(map[string]*Program),
	}

	//_ = "breakpoint"
	//runTC39File(tc39BASE, "test/language/types/number/8.5.1.js", t, ctx)
	//runTC39Tests(tc39BASE, "test/language", t, ctx)
	runTC39Tests(tc39BASE, "test/language/expressions", t, ctx)
	runTC39Tests(tc39BASE, "test/language/arguments-object", t, ctx)
	runTC39Tests(tc39BASE, "test/language/asi", t, ctx)
	runTC39Tests(tc39BASE, "test/language/directive-prologue", t, ctx)
	runTC39Tests(tc39BASE, "test/language/function-code", t, ctx)
	runTC39Tests(tc39BASE, "test/language/eval-code", t, ctx)
	runTC39Tests(tc39BASE, "test/language/global-code", t, ctx)
	runTC39Tests(tc39BASE, "test/language/identifier-resolution", t, ctx)
	runTC39Tests(tc39BASE, "test/language/identifiers", t, ctx)
	//runTC39Tests(tc39BASE, "test/language/literals", t, ctx) // octal sequences in strict mode
	runTC39Tests(tc39BASE, "test/language/punctuators", t, ctx)
	runTC39Tests(tc39BASE, "test/language/reserved-words", t, ctx)
	runTC39Tests(tc39BASE, "test/language/source-text", t, ctx)
	runTC39Tests(tc39BASE, "test/language/statements", t, ctx)
	runTC39Tests(tc39BASE, "test/language/types", t, ctx)
	runTC39Tests(tc39BASE, "test/language/white-space", t, ctx)
	runTC39Tests(tc39BASE, "test/built-ins", t, ctx)
	runTC39Tests(tc39BASE, "test/annexB/built-ins/String/prototype/substr", t, ctx)
}
