package goja

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

type HostResolveImportedModuleFunc func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error)

// TODO most things here probably should be unexported and names should be revised before merged in master
// Record should probably be dropped from everywhere

// ModuleRecord is the common interface for module record as defined in the EcmaScript specification
type ModuleRecord interface {
	// TODO move to ModuleInstance
	GetExportedNames(resolveset ...ModuleRecord) []string // TODO maybe this parameter is wrong
	ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool)
	Link() error
	Evaluate(*Runtime) (ModuleInstance, error)
}

type CyclicModuleRecordStatus uint8

const (
	Unlinked CyclicModuleRecordStatus = iota
	Linking
	Linked
	Evaluating
	Evaluated
)

type CyclicModuleRecord interface {
	ModuleRecord
	RequestedModules() []string
	InitializeEnvironment() error
	Instantiate() CyclicModuleInstance // TODO maybe should be taking the runtime
}

type LinkedSourceModuleRecord struct{}

func (c *compiler) CyclicModuleRecordConcreteLink(module ModuleRecord) error {
	stack := []CyclicModuleRecord{}
	if _, err := c.innerModuleLinking(newLinkState(), module, &stack, 0); err != nil {
		return err
	}
	return nil
}

type linkState struct {
	status           map[ModuleRecord]CyclicModuleRecordStatus
	dfsIndex         map[ModuleRecord]uint
	dfsAncestorIndex map[ModuleRecord]uint
}

func newLinkState() *linkState {
	return &linkState{
		status:           make(map[ModuleRecord]CyclicModuleRecordStatus),
		dfsIndex:         make(map[ModuleRecord]uint),
		dfsAncestorIndex: make(map[ModuleRecord]uint),
	}
}

type evaluationState struct {
	status           map[ModuleInstance]CyclicModuleRecordStatus
	dfsIndex         map[ModuleInstance]uint
	dfsAncestorIndex map[ModuleInstance]uint
}

func newEvaluationState() *evaluationState {
	return &evaluationState{
		status:           make(map[ModuleInstance]CyclicModuleRecordStatus),
		dfsIndex:         make(map[ModuleInstance]uint),
		dfsAncestorIndex: make(map[ModuleInstance]uint),
	}
}

func (c *compiler) innerModuleLinking(state *linkState, m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var module CyclicModuleRecord
	var ok bool
	if module, ok = m.(CyclicModuleRecord); !ok {
		err := m.Link() // TODO fix
		return index, err
	}
	if status := state.status[module]; status == Linking || status == Linked || status == Evaluated {
		return index, nil
	} else if status != Unlinked {
		return 0, errors.New("bad status on link") // TODO fix
	}
	state.status[module] = Linking
	state.dfsIndex[module] = index
	state.dfsAncestorIndex[module] = index
	index++
	*stack = append(*stack, module)
	var err error
	var requiredModule ModuleRecord
	for _, required := range module.RequestedModules() {
		requiredModule, err = c.hostResolveImportedModule(module, required)
		if err != nil {
			return 0, err
		}
		index, err = c.innerModuleLinking(state, requiredModule, stack, index)
		if err != nil {
			return 0, err
		}
		if requiredC, ok := requiredModule.(CyclicModuleRecord); ok {
			if state.status[requiredC] == Linking {
				if ancestorIndex := state.dfsAncestorIndex[module]; state.dfsAncestorIndex[requiredC] > ancestorIndex {
					state.dfsAncestorIndex[requiredC] = ancestorIndex
				}
			}
		}
	}
	err = module.InitializeEnvironment()
	if err != nil {
		return 0, err
	}
	if state.dfsAncestorIndex[module] == state.dfsIndex[module] {
		for i := len(*stack) - 1; i >= 0; i-- {
			requiredModule := (*stack)[i]
			*stack = (*stack)[:i]
			state.status[requiredModule] = Linked
			if requiredModule == module {
				break
			}
		}
	}
	return index, nil
}

func (r *Runtime) CyclicModuleRecordEvaluate(c ModuleRecord, resolve HostResolveImportedModuleFunc,
) (mi ModuleInstance, err error) {
	if r.modules == nil {
		r.modules = make(map[ModuleRecord]ModuleInstance)
	}
	stackInstance := []CyclicModuleInstance{}
	if mi, _, err = r.innerModuleEvaluation(newEvaluationState(), c, &stackInstance, 0, resolve); err != nil {
		return nil, err
	}

	return mi, nil
}

func (r *Runtime) innerModuleEvaluation(
	state *evaluationState,
	m ModuleRecord, stack *[]CyclicModuleInstance, index uint,
	resolve HostResolveImportedModuleFunc,
) (mi ModuleInstance, idx uint, err error) {
	if len(*stack) > 100000 {
		panic("too deep dependancy stack of 100000")
	}
	var cr CyclicModuleRecord
	var ok bool
	var c CyclicModuleInstance
	if cr, ok = m.(CyclicModuleRecord); !ok {
		mi, err = m.Evaluate(r)
		r.modules[m] = mi
		return mi, index, err
	} else {
		mi, ok = r.modules[m]
		if ok {
			return mi, index, nil
		}
		c = cr.Instantiate()
		mi = c
		r.modules[m] = c
	}
	if status := state.status[mi]; status == Evaluated {
		return nil, index, nil
	} else if status == Evaluating {
		return nil, index, nil
	}
	state.status[mi] = Evaluating
	state.dfsIndex[mi] = index
	state.dfsAncestorIndex[mi] = index
	index++

	*stack = append(*stack, c)
	var requiredModule ModuleRecord
	for _, required := range c.RequestedModules() {
		requiredModule, err = resolve(m, required)
		if err != nil {
			return nil, 0, err
		}
		var requiredInstance ModuleInstance
		requiredInstance, index, err = r.innerModuleEvaluation(state, requiredModule, stack, index, resolve)
		if err != nil {
			return nil, 0, err
		}
		if requiredC, ok := requiredInstance.(CyclicModuleInstance); ok {
			if state.status[requiredC] == Evaluating {
				if ancestorIndex := state.dfsAncestorIndex[c]; state.dfsAncestorIndex[requiredC] > ancestorIndex {
					state.dfsAncestorIndex[requiredC] = ancestorIndex
				}
			}
		}
	}
	mi, err = c.ExecuteModule(r)
	if err != nil {
		return nil, 0, err
	}

	if state.dfsAncestorIndex[c] == state.dfsIndex[c] {
		for i := len(*stack) - 1; i >= 0; i-- {
			requiredModuleInstance := (*stack)[i]
			*stack = (*stack)[:i]
			state.status[requiredModuleInstance] = Evaluated
			if requiredModuleInstance == c {
				break
			}
		}
	}
	return mi, index, nil
}

type (
	ModuleInstance interface {
		GetBindingValue(unistring.String) Value
	}
	CyclicModuleInstance interface {
		ModuleInstance
		RequestedModules() []string
		ExecuteModule(*Runtime) (ModuleInstance, error)
	}
)

var _ CyclicModuleRecord = &SourceTextModuleRecord{}

var _ CyclicModuleInstance = &SourceTextModuleInstance{}

type SourceTextModuleInstance struct {
	cyclicModuleStub
	moduleRecord *SourceTextModuleRecord
	// TODO figure out omething less idiotic
	exportGetters map[unistring.String]func() Value
}

func (s *SourceTextModuleInstance) ExecuteModule(rt *Runtime) (ModuleInstance, error) {
	_, err := rt.RunProgram(s.moduleRecord.p)
	return s, err
}

func (s *SourceTextModuleInstance) GetBindingValue(name unistring.String) Value {
	getter, ok := s.exportGetters[name]
	if !ok { // let's not panic in case somebody asks for a binding that isn't exported
		return nil
	}
	return getter()
}

type SourceTextModuleRecord struct {
	cyclicModuleStub
	body *ast.Program
	p    *Program
	// context
	// importmeta
	importEntries         []importEntry
	localExportEntries    []exportEntry
	indirectExportEntries []exportEntry
	starExportEntries     []exportEntry

	hostResolveImportedModule HostResolveImportedModuleFunc
}

type importEntry struct {
	moduleRequest string
	importName    string
	localName     string
	offset        int
}

type exportEntry struct {
	exportName    string
	moduleRequest string
	importName    string
	localName     string

	// no standard
	lex bool
}

func importEntriesFromAst(declarations []*ast.ImportDeclaration) ([]importEntry, error) {
	var result []importEntry
	names := make(map[string]struct{}, len(declarations))
	for _, importDeclarion := range declarations {
		importClause := importDeclarion.ImportClause
		if importDeclarion.FromClause == nil {
			continue // no entry in this case
		}
		moduleRequest := importDeclarion.FromClause.ModuleSpecifier.String()
		if named := importClause.NamedImports; named != nil {
			for _, el := range named.ImportsList {
				localName := el.Alias.String()
				if localName == "" {
					localName = el.IdentifierName.String()
				}
				if _, ok := names[localName]; ok {
					return nil, fmt.Errorf("duplicate bounded name %s", localName)
				}
				names[localName] = struct{}{}
				result = append(result, importEntry{
					moduleRequest: moduleRequest,
					importName:    el.IdentifierName.String(),
					localName:     localName,
					offset:        int(importDeclarion.Idx0()),
				})
			}
		}
		if def := importClause.ImportedDefaultBinding; def != nil {
			localName := def.Name.String()
			if _, ok := names[localName]; ok {
				return nil, fmt.Errorf("duplicate bounded name %s", localName)
			}
			names[localName] = struct{}{}
			result = append(result, importEntry{
				moduleRequest: moduleRequest,
				importName:    "default",
				localName:     localName,
				offset:        int(importDeclarion.Idx0()),
			})
		}
		if namespace := importClause.NameSpaceImport; namespace != nil {
			localName := namespace.ImportedBinding.String()
			if _, ok := names[localName]; ok {
				return nil, fmt.Errorf("duplicate bounded name %s", localName)
			}
			names[localName] = struct{}{}
			result = append(result, importEntry{
				moduleRequest: moduleRequest,
				importName:    "*",
				localName:     namespace.ImportedBinding.String(),
				offset:        int(importDeclarion.Idx0()),
			})
		}
	}
	return result, nil
}

func exportEntriesFromAst(declarations []*ast.ExportDeclaration) []exportEntry {
	var result []exportEntry
	for _, exportDeclaration := range declarations {
		if exportDeclaration.ExportFromClause != nil {
			exportFromClause := exportDeclaration.ExportFromClause
			if namedExports := exportFromClause.NamedExports; namedExports != nil {
				for _, spec := range namedExports.ExportsList {
					result = append(result, exportEntry{
						localName:  spec.IdentifierName.String(),
						exportName: spec.Alias.String(),
					})
				}
			} else if exportFromClause.IsWildcard {
				if from := exportDeclaration.FromClause; from != nil {
					result = append(result, exportEntry{
						exportName:    exportFromClause.Alias.String(),
						importName:    "*",
						moduleRequest: from.ModuleSpecifier.String(),
					})
				} else {
					result = append(result, exportEntry{
						exportName: exportFromClause.Alias.String(),
						importName: "*",
					})
				}
			} else {
				panic("wat")
			}
		} else if variableDeclaration := exportDeclaration.Variable; variableDeclaration != nil {
			for _, l := range variableDeclaration.List {
				id, ok := l.Target.(*ast.Identifier)
				if !ok {
					panic("target wasn;t identifier")
				}
				result = append(result, exportEntry{
					localName:  id.Name.String(),
					exportName: id.Name.String(),
					lex:        false,
				})

			}
		} else if LexicalDeclaration := exportDeclaration.LexicalDeclaration; LexicalDeclaration != nil {
			for _, l := range LexicalDeclaration.List {

				id, ok := l.Target.(*ast.Identifier)
				if !ok {
					panic("target wasn;t identifier")
				}
				result = append(result, exportEntry{
					localName:  id.Name.String(),
					exportName: id.Name.String(),
					lex:        true,
				})

			}
		} else if hoistable := exportDeclaration.HoistableDeclaration; hoistable != nil {
			localName := "default"
			exportName := "default"
			if hoistable.FunctionDeclaration != nil {
				if hoistable.FunctionDeclaration.Function.Name != nil {
					localName = string(hoistable.FunctionDeclaration.Function.Name.Name.String())
				}
			}
			if !exportDeclaration.IsDefault {
				exportName = localName
			}
			result = append(result, exportEntry{
				localName:  localName,
				exportName: exportName,
				lex:        true,
			})
		} else if fromClause := exportDeclaration.FromClause; fromClause != nil {
			if namedExports := exportDeclaration.NamedExports; namedExports != nil {
				for _, spec := range namedExports.ExportsList {
					alias := spec.IdentifierName.String()
					if spec.Alias.String() != "" { // TODO fix
						alias = spec.Alias.String()
					}
					result = append(result, exportEntry{
						importName:    spec.IdentifierName.String(),
						exportName:    alias,
						moduleRequest: fromClause.ModuleSpecifier.String(),
					})
				}
			} else {
				panic("wat")
			}
		} else if namedExports := exportDeclaration.NamedExports; namedExports != nil {
			for _, spec := range namedExports.ExportsList {
				alias := spec.IdentifierName.String()
				if spec.Alias.String() != "" { // TODO fix
					alias = spec.Alias.String()
				}
				result = append(result, exportEntry{
					localName:  spec.IdentifierName.String(),
					exportName: alias,
				})
			}
		} else if exportDeclaration.AssignExpression != nil {
			result = append(result, exportEntry{
				exportName: "default",
				localName:  "default",
				lex:        true,
			})
		} else if exportDeclaration.ClassDeclaration != nil {
			cls := exportDeclaration.ClassDeclaration.Class
			if exportDeclaration.IsDefault {
				result = append(result, exportEntry{
					exportName: "default",
					localName:  "default",
					lex:        true,
				})
			} else {
				result = append(result, exportEntry{
					exportName: cls.Name.Name.String(),
					localName:  cls.Name.Name.String(),
					lex:        true,
				})
			}
		} else {
			panic("wat")
		}
	}
	return result
}

func requestedModulesFromAst(statements []ast.Statement) []string {
	var result []string
	for _, st := range statements {
		switch imp := st.(type) {
		case *ast.ImportDeclaration:
			if imp.FromClause != nil {
				result = append(result, imp.FromClause.ModuleSpecifier.String())
			} else {
				result = append(result, imp.ModuleSpecifier.String())
			}
		case *ast.ExportDeclaration:
			if imp.FromClause != nil {
				result = append(result, imp.FromClause.ModuleSpecifier.String())
			}
		}
	}
	return result
}

func findImportByLocalName(importEntries []importEntry, name string) (importEntry, bool) {
	for _, i := range importEntries {
		if i.localName == name {
			return i, true
		}
	}

	return importEntry{}, false
}

// This should probably be part of Parse
// TODO arguments to this need fixing
func ParseModule(name, sourceText string, resolveModule HostResolveImportedModuleFunc, opts ...parser.Option) (*SourceTextModuleRecord, error) {
	// TODO asserts
	opts = append(opts, parser.IsModule)
	body, err := Parse(name, sourceText, opts...)
	_ = body
	if err != nil {
		return nil, err
	}
	return ModuleFromAST(body, resolveModule)
}

func ModuleFromAST(body *ast.Program, resolveModule HostResolveImportedModuleFunc) (*SourceTextModuleRecord, error) {
	requestedModules := requestedModulesFromAst(body.Body)
	importEntries, err := importEntriesFromAst(body.ImportEntries)
	if err != nil {
		// TODO create a separate error type
		return nil, &CompilerSyntaxError{CompilerError: CompilerError{
			Message: err.Error(),
		}}
	}
	// 6. Let importedBoundNames be ImportedLocalNames(importEntries).
	// ^ is skipped as we don't need it.

	var indirectExportEntries []exportEntry
	var localExportEntries []exportEntry
	var starExportEntries []exportEntry
	exportEntries := exportEntriesFromAst(body.ExportEntries)
	for _, ee := range exportEntries {
		if ee.moduleRequest == "" { // technically nil
			if ie, ok := findImportByLocalName(importEntries, ee.localName); !ok {
				localExportEntries = append(localExportEntries, ee)
			} else {
				if ie.importName == "*" {
					localExportEntries = append(localExportEntries, ee)
				} else {
					indirectExportEntries = append(indirectExportEntries, exportEntry{
						moduleRequest: ie.moduleRequest,
						importName:    ie.importName,
						exportName:    ee.exportName,
					})
				}
			}
		} else {
			if ee.importName == "*" && ee.exportName == "" {
				starExportEntries = append(starExportEntries, ee)
			} else {
				indirectExportEntries = append(indirectExportEntries, ee)
			}
		}
	}

	s := &SourceTextModuleRecord{
		// realm isn't implement
		// environment is undefined
		// namespace is undefined
		cyclicModuleStub: cyclicModuleStub{
			requestedModules: requestedModules,
		},
		// hostDefined TODO
		body: body,
		// Context empty
		// importMeta empty
		importEntries:         importEntries,
		localExportEntries:    localExportEntries,
		indirectExportEntries: indirectExportEntries,
		starExportEntries:     starExportEntries,

		hostResolveImportedModule: resolveModule,
	}

	names := s.getExportedNamesWithotStars() // we use this as the other one loops but wee need to early errors here
	sort.Strings(names)
	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			return nil, &CompilerSyntaxError{
				CompilerError: CompilerError{
					Message: fmt.Sprintf("Duplicate export name %s", names[i]),
				},
			}
		}
		// TODO other checks
	}

	return s, nil
}

func (module *SourceTextModuleRecord) getExportedNamesWithotStars() []string {
	exportedNames := make([]string, 0, len(module.localExportEntries)+len(module.indirectExportEntries))
	for _, e := range module.localExportEntries {
		exportedNames = append(exportedNames, e.exportName)
	}
	for _, e := range module.indirectExportEntries {
		exportedNames = append(exportedNames, e.exportName)
	}
	return exportedNames
}

func (module *SourceTextModuleRecord) GetExportedNames(exportStarSet ...ModuleRecord) []string {
	for _, el := range exportStarSet {
		if el == module { // better check
			// TODO assert
			return nil
		}
	}
	exportStarSet = append(exportStarSet, module)
	var exportedNames []string
	for _, e := range module.localExportEntries {
		exportedNames = append(exportedNames, e.exportName)
	}
	for _, e := range module.indirectExportEntries {
		exportedNames = append(exportedNames, e.exportName)
	}
	for _, e := range module.starExportEntries {
		requestedModule, err := module.hostResolveImportedModule(module, e.moduleRequest)
		if err != nil {
			panic(err)
		}
		starNames := requestedModule.GetExportedNames(exportStarSet...)

		for _, n := range starNames {
			if n != "default" {
				// TODO check if n i exportedNames and don't include it
				exportedNames = append(exportedNames, n)
			}
		}
	}

	return exportedNames
}

func (module *SourceTextModuleRecord) InitializeEnvironment() (err error) {
	c := newCompiler()
	defer func() {
		if x := recover(); x != nil {
			switch x1 := x.(type) {
			case *CompilerSyntaxError:
				err = x1
			default:
				panic(x)
			}
		}
	}()

	c.compileModule(module)
	module.p = c.p
	return
}

type ResolveSetElement struct {
	Module     ModuleRecord
	ExportName string
}

type ResolvedBinding struct {
	Module      ModuleRecord
	BindingName string
}

// GetModuleInstance returns an instance of an already instanciated module.
// If the ModuleRecord was not instanciated at this time it will return nil
func (r *Runtime) GetModuleInstance(m ModuleRecord) ModuleInstance {
	return r.modules[m]
}

func (module *SourceTextModuleRecord) ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool) {
	// TODO this whole algorithm can likely be used for not source module records a well
	if exportName == "" {
		panic("wat")
	}
	for _, r := range resolveset {
		if r.Module == module && exportName == r.ExportName { // TODO better
			return nil, false
		}
	}
	resolveset = append(resolveset, ResolveSetElement{Module: module, ExportName: exportName})
	for _, e := range module.localExportEntries {
		if exportName == e.exportName {
			// ii. ii. Return ResolvedBinding Record { [[Module]]: module, [[BindingName]]: e.[[LocalName]] }.
			return &ResolvedBinding{
				Module:      module,
				BindingName: e.localName,
			}, false
		}
	}

	for _, e := range module.indirectExportEntries {
		if exportName == e.exportName {
			importedModule, err := module.hostResolveImportedModule(module, e.moduleRequest)
			if err != nil {
				panic(err) // TODO return err
			}
			if e.importName == "*" {
				// 2. 2. Return ResolvedBinding Record { [[Module]]: importedModule, [[BindingName]]: "*namespace*" }.
				return &ResolvedBinding{
					Module:      importedModule,
					BindingName: "*namespace*",
				}, false
			} else {
				return importedModule.ResolveExport(e.importName, resolveset...)
			}
		}
	}
	if exportName == "default" {
		// This actually should've been caught above, but as it didn't it actually makes it s so the `default` export
		// doesn't resolve anything that is `export * ...`
		return nil, false
	}
	var starResolution *ResolvedBinding

	for _, e := range module.starExportEntries {
		importedModule, err := module.hostResolveImportedModule(module, e.moduleRequest)
		if err != nil {
			panic(err) // TODO return err
		}
		resolution, ambiguous := importedModule.ResolveExport(exportName, resolveset...)
		if ambiguous {
			return nil, true
		}
		if resolution != nil {
			if starResolution == nil {
				starResolution = resolution
			} else if resolution.Module != starResolution.Module || resolution.BindingName != starResolution.BindingName {
				return nil, true
			}
		}
	}
	return starResolution, false
}

func (module *SourceTextModuleRecord) Instantiate() CyclicModuleInstance {
	return &SourceTextModuleInstance{
		cyclicModuleStub: cyclicModuleStub{
			requestedModules: module.requestedModules,
		},
		moduleRecord:  module,
		exportGetters: make(map[unistring.String]func() Value),
	}
}

func (module *SourceTextModuleRecord) Evaluate(rt *Runtime) (ModuleInstance, error) {
	return rt.CyclicModuleRecordEvaluate(module, module.hostResolveImportedModule)
}

func (module *SourceTextModuleRecord) Link() error {
	c := newCompiler()
	c.hostResolveImportedModule = module.hostResolveImportedModule
	return c.CyclicModuleRecordConcreteLink(module)
}

type cyclicModuleStub struct {
	requestedModules []string
}

func (c *cyclicModuleStub) SetRequestedModules(modules []string) {
	c.requestedModules = modules
}

func (c *cyclicModuleStub) RequestedModules() []string {
	return c.requestedModules
}

type namespaceObject struct {
	baseObject
	m            ModuleRecord
	exports      map[unistring.String]struct{}
	exportsNames []unistring.String
}

func (r *Runtime) NamespaceObjectFor(m ModuleRecord) *Object {
	if r.moduleNamespaces == nil {
		r.moduleNamespaces = make(map[ModuleRecord]*namespaceObject)
	}
	if o, ok := r.moduleNamespaces[m]; ok {
		return o.val
	}
	o := r.createNamespaceObject(m)
	r.moduleNamespaces[m] = o
	return o.val
}

func (r *Runtime) createNamespaceObject(m ModuleRecord) *namespaceObject {
	o := &Object{runtime: r}
	no := &namespaceObject{m: m}
	no.val = o
	no.extensible = true
	no.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
		Value: newStringValue("Module"),
	}, true)
	no.extensible = false
	o.self = no
	no.init()
	no.exports = make(map[unistring.String]struct{})

	for _, exportName := range m.GetExportedNames() {
		v, ambiguous := no.m.ResolveExport(exportName)
		if ambiguous || v == nil {
			continue
		}
		no.exports[unistring.NewFromString(exportName)] = struct{}{}
		no.exportsNames = append(no.exportsNames, unistring.NewFromString(exportName))
	}
	return no
}

func (no *namespaceObject) stringKeys(all bool, accum []Value) []Value {
	for name := range no.exports {
		if !all { //  TODO this seems off
			_ = no.getOwnPropStr(name)
		}
		accum = append(accum, stringValueFromRaw(name))
	}
	// TODO optimize thsi
	sort.Slice(accum, func(i, j int) bool {
		return accum[i].String() < accum[j].String()
	})
	return accum
}

type namespacePropIter struct {
	no  *namespaceObject
	idx int
}

func (no *namespaceObject) iterateStringKeys() iterNextFunc {
	return (&namespacePropIter{
		no: no,
	}).next
}

func (no *namespaceObject) iterateKeys() iterNextFunc {
	return no.iterateStringKeys()
}

func (i *namespacePropIter) next() (propIterItem, iterNextFunc) {
	for i.idx < len(i.no.exportsNames) {
		name := i.no.exportsNames[i.idx]
		i.idx++
		prop := i.no.getOwnPropStr(name)
		if prop != nil {
			return propIterItem{name: stringValueFromRaw(name), value: prop}, i.next
		}
	}
	return propIterItem{}, nil
}

func (no *namespaceObject) getOwnPropStr(name unistring.String) Value {
	if _, ok := no.exports[name]; !ok {
		return nil
	}
	v, ambiguous := no.m.ResolveExport(name.String())
	if ambiguous || v == nil {
		no.val.runtime.throwReferenceError((name))
	}
	if v.BindingName == "*namespace*" {
		return &valueProperty{
			value:        no.val.runtime.NamespaceObjectFor(v.Module),
			writable:     true,
			configurable: false,
			enumerable:   true,
		}
	}

	mi := no.val.runtime.modules[v.Module]
	b := mi.GetBindingValue(unistring.NewFromString(v.BindingName))
	if b == nil {
		// TODO figure this out - this is needed as otherwise valueproperty is thought to not have a value
		// which isn't really correct in a particular test around isFrozen
		b = _null
	}
	return &valueProperty{
		value:        b,
		writable:     true,
		configurable: false,
		enumerable:   true,
	}
}

func (no *namespaceObject) hasOwnPropertyStr(name unistring.String) bool {
	_, ok := no.exports[name]
	return ok
}

func (no *namespaceObject) getStr(name unistring.String, receiver Value) Value {
	prop := no.getOwnPropStr(name)
	if prop, ok := prop.(*valueProperty); ok {
		if receiver == nil {
			return prop.get(no.val)
		}
		return prop.get(receiver)
	}
	return prop
}

func (no *namespaceObject) setOwnStr(name unistring.String, val Value, throw bool) bool {
	no.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
	return false
}

func (no *namespaceObject) deleteStr(name unistring.String, throw bool) bool {
	if _, exists := no.exports[name]; exists {
		no.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		return false
	}
	return true
}

func (no *namespaceObject) defineOwnPropertyStr(name unistring.String, desc PropertyDescriptor, throw bool) bool {
	returnFalse := func() bool {
		var buf bytes.Buffer
		for _, stack := range no.val.runtime.CaptureCallStack(0, nil) {
			stack.Write(&buf)
			buf.WriteRune('\n')
		}
		if throw {
			no.val.runtime.typeErrorResult(throw, "Cannot add property %s, object is not extensible", name)
		}
		return false
	}
	if !no.hasOwnPropertyStr(name) {
		return returnFalse()
	}
	if desc.Empty() {
		return true
	}
	if desc.Writable == FLAG_FALSE {
		return returnFalse()
	}
	if desc.Configurable == FLAG_TRUE {
		return returnFalse()
	}
	if desc.Enumerable == FLAG_FALSE {
		return returnFalse()
	}
	if desc.Value != nil && desc.Value != no.getOwnPropStr(name) {
		return returnFalse()
	}
	// TODO more checks
	return true
}
