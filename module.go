package goja

import (
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
	GetExportedNames(resolveset ...*SourceTextModuleRecord) []string // TODO maybe this parameter is wrong
	ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool)
	Link() error
	Evaluate(*Runtime) (ModuleInstance, error)
	/*
		Namespace() *Namespace
		SetNamespace(*Namespace)
	*/
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
	// TODO this probably shouldn't really be an interface ... or at least the current one is quite bad and big
	ModuleRecord
	Status() CyclicModuleRecordStatus
	SetStatus(CyclicModuleRecordStatus)
	EvaluationError() error
	SetEvaluationError(error)
	DFSIndex() uint
	SetDFSIndex(uint)
	DFSAncestorIndex() uint
	SetDFSAncestorIndex(uint)
	RequestedModules() []string
	InitializeEnvorinment() error
	ExecuteModule(*Runtime) (Value, error)
	Instanciate() CyclicModuleInstance
}

type LinkedSourceModuleRecord struct{}

func (c *compiler) CyclicModuleRecordConcreteLink(module CyclicModuleRecord) error {
	if module.Status() == Linking || module.Status() == Evaluating {
		return fmt.Errorf("bad status %+v on link", module.Status())
	}

	stack := []CyclicModuleRecord{}
	if _, err := c.innerModuleLinking(module, &stack, 0); err != nil {
		fmt.Println(err)
		for _, m := range stack {
			if m.Status() != Linking {
				return fmt.Errorf("bad status %+v on link", m.Status())
			}
			m.SetStatus(Unlinked)

			// TODO reset the rest

		}
		module.SetStatus(Unlinked)
		return err

	}
	return nil
}

func (c *compiler) innerModuleLinking(m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var module CyclicModuleRecord
	var ok bool
	if module, ok = m.(CyclicModuleRecord); !ok {
		err := m.Link() // TODO fix
		return index, err
	}
	if status := module.Status(); status == Linking || status == Linked || status == Evaluated {
		return index, nil
	} else if status != Unlinked {
		return 0, errors.New("bad status on link") // TODO fix
	}
	module.SetStatus(Linking)
	module.SetDFSIndex(index)
	module.SetDFSAncestorIndex(index)
	index++
	*stack = append(*stack, module)
	var err error
	var requiredModule ModuleRecord
	for _, required := range module.RequestedModules() {
		requiredModule, err = c.hostResolveImportedModule(module, required)
		if err != nil {
			return 0, err
		}
		index, err = c.innerModuleLinking(requiredModule, stack, index)
		if err != nil {
			return 0, err
		}
		if requiredC, ok := requiredModule.(CyclicModuleRecord); ok {
			// TODO some asserts
			if requiredC.Status() == Linking {
				if ancestorIndex := module.DFSAncestorIndex(); requiredC.DFSAncestorIndex() > ancestorIndex {
					requiredC.SetDFSAncestorIndex(ancestorIndex)
				}
			}
		}
	}
	err = module.InitializeEnvorinment()
	if err != nil {
		return 0, err
	}
	// TODO more asserts

	if module.DFSAncestorIndex() == module.DFSIndex() {
		for i := len(*stack) - 1; i >= 0; i-- {
			requiredModule := (*stack)[i]
			// TODO assert
			requiredModule.SetStatus(Linked)
			if requiredModule == module {
				break
			}
		}
	}
	return index, nil
}

func (r *Runtime) CyclicModuleRecordEvaluate(c CyclicModuleRecord, name string, resolve HostResolveImportedModuleFunc,
) (mi ModuleInstance, err error) {
	// TODO asserts
	if r.modules == nil {
		r.modules = make(map[ModuleRecord]ModuleInstance)
	}
	stackInstance := []CyclicModuleInstance{}
	if mi, _, err = r.innerModuleEvaluation(c, &stackInstance, 0, name, resolve); err != nil {
		/*
			for _, m := range stack {
				// TODO asserts
				m.SetStatus(Evaluated)
				m.SetEvaluationError(err)
			}
		*/
		// TODO asserts
		return nil, err
	}

	// TODO asserts
	return mi, nil
}

func (r *Runtime) innerModuleEvaluation(
	m ModuleRecord, stack *[]CyclicModuleInstance, index uint,
	name string, resolve HostResolveImportedModuleFunc,
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
		mi = c
		c = cr.Instanciate()
		r.modules[m] = c
	}
	if status := c.Status(); status == Evaluated { // TODO switch
		return nil, index, c.EvaluationError()
	} else if status == Evaluating {
		return nil, index, nil
	} else if status != Linked {
		return nil, 0, errors.New("module isn't linked when it's being evaluated")
	}
	c.SetStatus(Evaluating)
	c.SetDFSIndex(index)
	c.SetDFSAncestorIndex(index)
	index++

	*stack = append(*stack, c)
	var requiredModule ModuleRecord
	for _, required := range c.RequestedModules() {
		requiredModule, err = resolve(c, required)
		if err != nil {
			return nil, 0, err
		}
		var requiredInstance ModuleInstance
		requiredInstance, index, err = r.innerModuleEvaluation(requiredModule, stack, index, required, resolve)
		if err != nil {
			return nil, 0, err
		}
		if requiredC, ok := requiredInstance.(CyclicModuleInstance); ok {
			// TODO some asserts
			if requiredC.Status() == Evaluating {
				if ancestorIndex := c.DFSAncestorIndex(); requiredC.DFSAncestorIndex() > ancestorIndex {
					requiredC.SetDFSAncestorIndex(ancestorIndex)
				}
			}
		}
	}
	mi, err = c.ExecuteModule(r)
	if err != nil {
		return nil, 0, err
	}
	// TODO asserts

	if c.DFSAncestorIndex() == c.DFSIndex() {
		for i := len(*stack) - 1; i >= 0; i-- {
			requiredModule := (*stack)[i]
			// TODO assert
			requiredModule.SetStatus(Evaluated)
			if requiredModule == c {
				break
			}
		}
	}
	return mi, index, nil
}

type (
	ModuleInstance interface {
		// Evaluate(rt *Runtime) (ModuleInstance, error)
		GetBindingValue(unistring.String, bool) Value
	}
	CyclicModuleInstance interface {
		ModuleInstance
		Status() CyclicModuleRecordStatus
		SetStatus(CyclicModuleRecordStatus)
		EvaluationError() error
		SetEvaluationError(error)
		DFSIndex() uint
		SetDFSIndex(uint)
		DFSAncestorIndex() uint
		SetDFSAncestorIndex(uint)
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

func (s *SourceTextModuleInstance) GetBindingValue(name unistring.String, b bool) Value {
	getter, ok := s.exportGetters[name]
	if !ok {
		return nil
		// panic(name + " is not defined, this shoukldn't be possible due to how ESM works")
	}
	return getter()
}

type SourceTextModuleRecord struct {
	cyclicModuleStub
	name string // TODO remove this :crossed_fingers:
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
}

func importEntriesFromAst(declarations []*ast.ImportDeclaration) []importEntry {
	var result []importEntry
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
				result = append(result, importEntry{
					moduleRequest: moduleRequest,
					importName:    el.IdentifierName.String(),
					localName:     localName,
					offset:        int(importDeclarion.Idx0()),
				})
			}
		}
		if def := importClause.ImportedDefaultBinding; def != nil {
			result = append(result, importEntry{
				moduleRequest: moduleRequest,
				importName:    "default",
				localName:     def.Name.String(),
				offset:        int(importDeclarion.Idx0()),
			})
		}
		if namespace := importClause.NameSpaceImport; namespace != nil {
			result = append(result, importEntry{
				moduleRequest: moduleRequest,
				importName:    "*",
				localName:     namespace.ImportedBinding.String(),
				offset:        int(importDeclarion.Idx0()),
			})
		}
	}
	return result
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
				})

			}
		} else if hoistable := exportDeclaration.HoistableDeclaration; hoistable != nil {
			localName := "*default*"
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
			})
		} else {
			panic("wat")
		}
	}
	return result
}

func requestedModulesFromAst(imports []*ast.ImportDeclaration, exports []*ast.ExportDeclaration) []string {
	var result []string
	for _, imp := range imports {
		if imp.FromClause != nil {
			result = append(result, imp.FromClause.ModuleSpecifier.String())
		} else {
			result = append(result, imp.ModuleSpecifier.String())
		}
	}
	for _, exp := range exports {
		if exp.FromClause != nil {
			result = append(result, exp.FromClause.ModuleSpecifier.String())
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
	return ModuleFromAST(name, body, resolveModule)
}

func ModuleFromAST(name string, body *ast.Program, resolveModule HostResolveImportedModuleFunc) (*SourceTextModuleRecord, error) {
	requestedModules := requestedModulesFromAst(body.ImportEntries, body.ExportEntries)
	importEntries := importEntriesFromAst(body.ImportEntries)
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
		name: name,
		// realm isn't implement
		// environment is undefined
		// namespace is undefined
		cyclicModuleStub: cyclicModuleStub{
			status:           Unlinked,
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

func (module *SourceTextModuleRecord) ExecuteModule(rt *Runtime) (Value, error) {
	// TODO copy runtime.RunProgram here with some changes so that it doesn't touch the global ?

	return rt.RunProgram(module.p)
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

func (module *SourceTextModuleRecord) GetExportedNames(exportStarSet ...*SourceTextModuleRecord) []string {
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

func (module *SourceTextModuleRecord) InitializeEnvorinment() (err error) {
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

/*
func (rt *Runtime) getModuleNamespace(module ModuleRecord) *Namespace {
	if c, ok := module.(CyclicModuleRecord); ok && c.Status() == Unlinked {
		panic("oops") // TODO beter oops
	}
	namespace := module.Namespace()
	if namespace == nil {
		exportedNames := module.GetExportedNames()
		var unambiguousNames []string
		for _, name := range exportedNames {
			_, ok := module.ResolveExport(name)
			if ok {
				unambiguousNames = append(unambiguousNames, name)
			}
		}
		namespace := rt.moduleNamespaceCreate(module, unambiguousNames)
		module.SetNamespace(namespace)
	}
	return namespace
}

// TODO this probably should really be goja.Object
type Namespace struct {
	module  ModuleRecord
	exports []string
}

func (rt *Runtime) moduleNamespaceCreate(module ModuleRecord, exports []string) *Namespace {
	sort.Strings(exports)
	return &Namespace{
		module:  module,
		exports: exports,
	}
}

*/
type ResolveSetElement struct {
	Module     ModuleRecord
	ExportName string
}

type ResolvedBinding struct {
	Module      ModuleRecord
	BindingName string
}

func (module *SourceTextModuleRecord) ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool) {
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

func (module *SourceTextModuleRecord) Instanciate() CyclicModuleInstance {
	return &SourceTextModuleInstance{
		cyclicModuleStub: cyclicModuleStub{
			status:           module.status,
			requestedModules: module.requestedModules,
		},
		moduleRecord:  module,
		exportGetters: make(map[unistring.String]func() Value),
	}
}

func (module *SourceTextModuleRecord) Evaluate(rt *Runtime) (ModuleInstance, error) {
	return rt.CyclicModuleRecordEvaluate(module, module.name, module.hostResolveImportedModule)
}

func (module *SourceTextModuleRecord) Link() error {
	c := newCompiler()
	c.hostResolveImportedModule = module.hostResolveImportedModule
	return c.CyclicModuleRecordConcreteLink(module)
}

type cyclicModuleStub struct {
	// namespace        *Namespace
	status           CyclicModuleRecordStatus
	dfsIndex         uint
	ancestorDfsIndex uint
	evaluationError  error
	requestedModules []string
}

func (c *cyclicModuleStub) SetStatus(status CyclicModuleRecordStatus) {
	c.status = status
}

func (c *cyclicModuleStub) Status() CyclicModuleRecordStatus {
	return c.status
}

func (c *cyclicModuleStub) SetDFSIndex(index uint) {
	c.dfsIndex = index
}

func (c *cyclicModuleStub) DFSIndex() uint {
	return c.dfsIndex
}

func (c *cyclicModuleStub) SetDFSAncestorIndex(index uint) {
	c.ancestorDfsIndex = index
}

func (c *cyclicModuleStub) DFSAncestorIndex() uint {
	return c.ancestorDfsIndex
}

func (c *cyclicModuleStub) SetEvaluationError(err error) {
	c.evaluationError = err
}

func (c *cyclicModuleStub) EvaluationError() error {
	return c.evaluationError
}

func (c *cyclicModuleStub) SetRequestedModules(modules []string) {
	c.requestedModules = modules
}

func (c *cyclicModuleStub) RequestedModules() []string {
	return c.requestedModules
}

/*
func (c *cyclicModuleStub) Namespace() *Namespace {
	return c.namespace
}

func (c *cyclicModuleStub) SetNamespace(namespace *Namespace) {
	c.namespace = namespace
}
*/
