package goja_test // this is on purpose in a separate package

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/dop251/goja/unistring"
)

type simpleComboResolver struct {
	cache        map[string]cacheElement
	reverseCache map[goja.ModuleRecord]string
	fs           fs.FS
	custom       func(interface{}, string) (goja.ModuleRecord, error)
}
type cacheElement struct {
	m   goja.ModuleRecord
	err error
}

func newSimpleComboResolver() *simpleComboResolver {
	return &simpleComboResolver{cache: make(map[string]cacheElement), reverseCache: make(map[goja.ModuleRecord]string)}
}

func (s *simpleComboResolver) resolve(referencingScriptOrModule interface{}, specifier string) (goja.ModuleRecord, error) {
	k, ok := s.cache[specifier]
	if ok {
		return k.m, k.err
	}
	if strings.HasPrefix(specifier, "custom:") {
		p, err := s.custom(referencingScriptOrModule, specifier)
		s.cache[specifier] = cacheElement{m: p, err: err}
		return p, err
	}
	b, err := fs.ReadFile(s.fs, specifier)
	if err != nil {
		s.cache[specifier] = cacheElement{err: err}
		return nil, err
	}
	p, err := goja.ParseModule(specifier, string(b), s.resolve)
	if err != nil {
		s.cache[specifier] = cacheElement{err: err}
		return nil, err
	}
	s.cache[specifier] = cacheElement{m: p}
	s.reverseCache[p] = specifier
	return p, nil
}

type unresolvedBinding struct {
	module  string
	bidning string
}

func TestNotSourceModulesBigTest(t *testing.T) {
	t.Parallel()
	resolver := newSimpleComboResolver()
	resolver.custom = func(_ interface{}, specifier string) (goja.ModuleRecord, error) {
		switch specifier {
		case "custom:coolstuff":
			return &simpleModuleImpl{}, nil
		case "custom:coolstuff2":
			return &cyclicModuleImpl{
				resolve:          resolver.resolve,
				requestedModules: []string{"custom:coolstuff3", "custom:coolstuff"},
				exports: map[string]unresolvedBinding{
					"coolStuff": {
						bidning: "coolStuff",
						module:  "custom:coolstuff",
					},
					"otherCoolStuff": { // request it from third module which will request it back from us
						bidning: "coolStuff",
						module:  "custom:coolstuff3",
					},
				},
			}, nil
		case "custom:coolstuff3":
			return &cyclicModuleImpl{
				resolve:          resolver.resolve,
				requestedModules: []string{"custom:coolstuff2"},
				exports: map[string]unresolvedBinding{
					"coolStuff": { // request it back from the module
						bidning: "coolStuff",
						module:  "custom:coolstuff2",
					},
				},
			}, nil
		default:
			return nil, fmt.Errorf("unknown module %q", specifier)
		}
	}
	mapfs := make(fstest.MapFS)
	mapfs["main.js"] = &fstest.MapFile{
		Data: []byte(`
        import {coolStuff} from "custom:coolstuff";
        import {coolStuff as coolStuff3, otherCoolStuff} from "custom:coolstuff2";
        if (coolStuff != 5) {
            throw "coolStuff isn't a 5 it is a "+ coolStuff
        }
        if (coolStuff3 != 5) {
            throw "coolStuff3 isn't a 5 it is a "+ coolStuff3
        }
        if (otherCoolStuff != 5) {
            throw "otherCoolStuff isn't a 5 it is a "+ otherCoolStuff
        }
        `),
	}
	resolver.fs = mapfs
	m, err := resolver.resolve(nil, "main.js")
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	p := m.(*goja.SourceTextModuleRecord)

	err = p.Link()
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	vm := goja.New()
	_, err = m.Evaluate(vm)
	if err != nil {
		t.Fatalf("got error %s", err)
	}
}

// START of simple module implementation
type simpleModuleImpl struct{}

var _ goja.ModuleRecord = &simpleModuleImpl{}

func (s *simpleModuleImpl) Link() error {
	// this does nothing on this
	return nil
}

func (s *simpleModuleImpl) ResolveExport(exportName string, resolveset ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	if exportName == "coolStuff" {
		return &goja.ResolvedBinding{
			BindingName: exportName,
			Module:      s,
		}, false
	}
	return nil, false
}

func (s *simpleModuleImpl) Evaluate(rt *goja.Runtime) (goja.ModuleInstance, error) {
	return &simpleModuleInstanceImpl{rt: rt}, nil
}

func (s *simpleModuleImpl) GetExportedNames(records ...goja.ModuleRecord) []string {
	return []string{"coolStuff"}
}

type simpleModuleInstanceImpl struct {
	rt *goja.Runtime
}

func (si *simpleModuleInstanceImpl) GetBindingValue(exportName unistring.String) goja.Value {
	if exportName.String() == "coolStuff" {
		return si.rt.ToValue(5)
	}
	return nil
}

// START of cyclic module implementation
type cyclicModuleImpl struct {
	requestedModules []string
	exports          map[string]unresolvedBinding
	resolve          goja.HostResolveImportedModuleFunc
}

var _ goja.CyclicModuleRecord = &cyclicModuleImpl{}

func (s *cyclicModuleImpl) InitializeEnvironment() error {
	return nil
}

func (s *cyclicModuleImpl) Instantiate() goja.CyclicModuleInstance {
	return &cyclicModuleInstanceImpl{module: s}
}

func (s *cyclicModuleImpl) RequestedModules() []string {
	return s.requestedModules
}

func (s *cyclicModuleImpl) Link() error {
	// this does nothing on this
	return nil
}

func (s *cyclicModuleImpl) Evaluate(rt *goja.Runtime) (goja.ModuleInstance, error) {
	return nil, nil
}

func (s *cyclicModuleImpl) ResolveExport(exportName string, resolveset ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	b, ok := s.exports[exportName]
	if !ok {
		return nil, false
	}

	m, err := s.resolve(s, b.module)
	if err != nil {
		panic(err)
	}

	return &goja.ResolvedBinding{
		Module:      m,
		BindingName: b.bidning,
	}, false
}

func (s *cyclicModuleImpl) GetExportedNames(records ...goja.ModuleRecord) []string {
	return []string{"coolStuff"}
}

type cyclicModuleInstanceImpl struct {
	rt     *goja.Runtime
	module *cyclicModuleImpl
}

func (si *cyclicModuleInstanceImpl) ExecuteModule(rt *goja.Runtime) (goja.ModuleInstance, error) {
	si.rt = rt
	// TODO others
	return nil, nil
}

func (si *cyclicModuleInstanceImpl) GetBindingValue(exportName unistring.String) goja.Value {
	b, ambigious := si.module.ResolveExport(exportName.String())
	if ambigious || b == nil {
		panic("fix this")
	}
	return si.rt.GetModuleInstance(b.Module).GetBindingValue(exportName)
}

func TestSourceMetaImport(t *testing.T) {
	t.Parallel()
	resolver := newSimpleComboResolver()
	mapfs := make(fstest.MapFS)
	mapfs["main.js"] = &fstest.MapFile{
		Data: []byte(`
        import { meta } from "b.js"

        if (meta.url != "file:///b.js") {
            throw "wrong url " + meta.url + " for b.js"
        }

        if (import.meta.url != "file:///main.js") {
            throw "wrong url " + import.meta.url + " for main.js"
        }
        `),
	}
	mapfs["b.js"] = &fstest.MapFile{
		Data: []byte(`
        export var meta = import.meta
        `),
	}
	resolver.fs = mapfs
	m, err := resolver.resolve(nil, "main.js")
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	p := m.(*goja.SourceTextModuleRecord)

	err = p.Link()
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	vm := goja.New()
	vm.SetGetImportMetaProperties(func(m goja.ModuleRecord) []goja.MetaProperty {
		specifier, ok := resolver.reverseCache[m]
		if !ok {
			panic("we got import.meta for module that wasn't imported")
		}
		return []goja.MetaProperty{
			{
				Key:   "url",
				Value: vm.ToValue("file:///" + specifier),
			},
		}
	})
	_, err = m.Evaluate(vm)
	if err != nil {
		t.Fatalf("got error %s", err)
	}
}
