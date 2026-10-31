package goja

import (
	"fmt"
	"sort"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

// ModuleStatus is a module record's lifecycle state.
type ModuleStatus int

const (
	ModuleUnlinked ModuleStatus = iota
	ModuleLinking
	ModuleLinked
	ModuleEvaluating
	ModuleEvaluated
)

// ModuleLoader resolves a module specifier relative to the referrer module.
type ModuleLoader func(specifier, referrer string) (string, error)

type moduleImportEntry struct {
	localName  string
	importName string
	from       string
}

type moduleExportEntry struct {
	localName  string
	exportName string
	from       string // non-empty for indirect exports
}

// ModuleRecord holds the compiled shape of a module.
type ModuleRecord struct {
	name      string
	program   *ast.Program
	status    ModuleStatus
	imports   []moduleImportEntry
	exports   []moduleExportEntry
	stars     []string
	starAs    map[string]string // namespace name -> source
	namespace *Object
}

// SetModuleLoader installs the host hook that resolves module specifiers.
func (r *Runtime) SetModuleLoader(l ModuleLoader) {
	r.moduleLoader = l
}

// DefineModule parses src as a module and registers it under name.
func (r *Runtime) DefineModule(name, src string) error {
	prog, err := parser.ParseModule(name, src)
	if err != nil {
		return err
	}
	rec := &ModuleRecord{name: name, program: prog, status: ModuleUnlinked, starAs: map[string]string{}}
	seen := map[string]bool{}
	for _, stmt := range prog.Body {
		switch s := stmt.(type) {
		case *ast.ImportDeclaration:
			if s.Source == nil {
				continue
			}
			from := s.Source.Value.String()
			for _, spec := range s.Specifiers {
				importName := "default"
				switch spec.Kind {
				case ast.ImportNamespace:
					importName = "*"
				case ast.ImportNamed:
					if spec.Imported != nil {
						importName = spec.Imported.Name.String()
					}
				}
				rec.imports = append(rec.imports, moduleImportEntry{
					localName:  spec.Local.Name.String(),
					importName: importName,
					from:       from,
				})
			}
		case *ast.ExportDeclaration:
			if s.Declaration != nil && !s.IsDefault {
				for _, name := range moduleDeclaredNames(s.Declaration) {
					if err := addExport(rec, seen, name, name, ""); err != nil {
						return err
					}
				}
			}
			if s.IsDefault {
				if err := addExport(rec, seen, "", "default", ""); err != nil {
					return err
				}
				continue
			}
			if s.Source != nil {
				from := s.Source.Value.String()
				if s.Namespace != nil {
					name := s.Namespace.Name.String()
					rec.stars = append(rec.stars, from)
					rec.starAs[name] = from
					if err := addExport(rec, seen, "", name, from); err != nil {
						return err
					}
					continue
				}
				if len(s.Specifiers) == 0 {
					rec.stars = append(rec.stars, from)
					continue
				}
				for _, spec := range s.Specifiers {
					local := ""
					if spec.Local != nil {
						local = spec.Local.Name.String()
					}
					exported := local
					if spec.Exported != nil {
						exported = spec.Exported.Name.String()
					}
					if err := addExport(rec, seen, local, exported, from); err != nil {
						return err
					}
				}
				continue
			}
			for _, spec := range s.Specifiers {
				local := ""
				if spec.Local != nil {
					local = spec.Local.Name.String()
				}
				exported := local
				if spec.Exported != nil {
					exported = spec.Exported.Name.String()
				}
				if err := addExport(rec, seen, local, exported, ""); err != nil {
					return err
				}
			}
		}
	}
	if r.modules == nil {
		r.modules = map[string]*ModuleRecord{}
	}
	r.modules[name] = rec
	return nil
}

func addExport(rec *ModuleRecord, seen map[string]bool, local, exported, from string) error {
	if exported == "" {
		return nil
	}
	if seen[exported] {
		return fmt.Errorf("duplicate export %q in module %q", exported, rec.name)
	}
	seen[exported] = true
	rec.exports = append(rec.exports, moduleExportEntry{localName: local, exportName: exported, from: from})
	return nil
}

// LinkModule resolves and links a registered module and its dependencies.
func (r *Runtime) LinkModule(name string) error {
	return r.linkModule(name)
}

func (r *Runtime) linkModule(name string) error {
	rec, ok := r.modules[name]
	if !ok {
		return fmt.Errorf("module %q is not defined", name)
	}
	switch rec.status {
	case ModuleLinking, ModuleLinked, ModuleEvaluating, ModuleEvaluated:
		return nil
	}
	rec.status = ModuleLinking
	resolveAndLink := func(spec string) error {
		dep := spec
		if r.moduleLoader != nil {
			resolved, err := r.moduleLoader(spec, name)
			if err != nil {
				return err
			}
			dep = resolved
		}
		return r.linkModule(dep)
	}
	for _, imp := range rec.imports {
		if err := resolveAndLink(imp.from); err != nil {
			return err
		}
	}
	for _, exp := range rec.exports {
		if exp.from != "" {
			if err := resolveAndLink(exp.from); err != nil {
				return err
			}
		}
	}
	for _, star := range rec.stars {
		if err := resolveAndLink(star); err != nil {
			return err
		}
	}
	rec.status = ModuleLinked
	return nil
}

// moduleDeclaredNames returns the names bound by an exported declaration.
func moduleDeclaredNames(n ast.Node) []string {
	switch d := n.(type) {
	case *ast.VariableStatement:
		return moduleBindingNames(d.List)
	case *ast.LexicalDeclaration:
		return moduleBindingNames(d.List)
	case *ast.FunctionDeclaration:
		if d.Function != nil && d.Function.Name != nil {
			return []string{d.Function.Name.Name.String()}
		}
	case *ast.ClassDeclaration:
		if d.Class != nil && d.Class.Name != nil {
			return []string{d.Class.Name.Name.String()}
		}
	}
	return nil
}

func moduleBindingNames(list []*ast.Binding) []string {
	var res []string
	for _, b := range list {
		if id, ok := b.Target.(*ast.Identifier); ok {
			res = append(res, id.Name.String())
		}
	}
	return res
}

// ModuleNamespace returns the module namespace exotic object for a module.
func (r *Runtime) ModuleNamespace(name string) (Value, error) {
	rec, ok := r.modules[name]
	if !ok {
		return nil, fmt.Errorf("module %q is not defined", name)
	}
	if rec.namespace != nil {
		return rec.namespace, nil
	}
	ns := r.newBaseObject(nil, classObject)
	seen := map[string]bool{}
	var names []string
	for _, e := range rec.exports {
		if e.exportName == "" || seen[e.exportName] {
			continue
		}
		seen[e.exportName] = true
		names = append(names, e.exportName)
	}
	sort.Strings(names)
	for _, n := range names {
		key := unistring.NewFromString(n)
		ns.defineOwnPropertyStr(key, PropertyDescriptor{
			Value:        _undefined,
			Writable:     FLAG_TRUE,
			Enumerable:   FLAG_TRUE,
			Configurable: FLAG_FALSE,
		}, false)
	}
	ns.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
		Value:        asciiString("Module"),
		Writable:     FLAG_FALSE,
		Enumerable:   FLAG_FALSE,
		Configurable: FLAG_FALSE,
	}, false)
	rec.namespace = ns.val
	return ns.val, nil
}
