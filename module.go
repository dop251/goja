package goja

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

// TODO most things here probably should be unexported and names should be revised before merged in master
// Record should probably be dropped from everywhere

// ModuleRecord is the common interface for module record as defined in the EcmaScript specification
type ModuleRecord interface {
	GetExportedNames(resolveset ...*SourceTextModuleRecord) []string // TODO maybe this parameter is wrong
	ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool)
	Link() error
	Evaluate() error
	Namespace() *Namespace
	SetNamespace(*Namespace)
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
	ExecuteModule() error
}

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

func (co *compiler) innerModuleLinking(m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var module CyclicModuleRecord
	var ok bool
	if module, ok = m.(CyclicModuleRecord); !ok {
		return index, m.Link()
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
		requiredModule, err = co.hostResolveImportedModule(module, required)
		if err != nil {
			return 0, err
		}
		index, err = co.innerModuleLinking(requiredModule, stack, index)
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
			requiredC := requiredModule.(CyclicModuleRecord)
			requiredC.SetStatus(Linked)
			if requiredC == module {
				break
			}
		}
	}
	return index, nil
}

func (rt *Runtime) CyclicModuleRecordEvaluate(c CyclicModuleRecord) error {
	// TODO asserts
	stack := []CyclicModuleRecord{}
	if _, err := rt.innerModuleEvaluation(c, &stack, 0); err != nil {

		for _, m := range stack {
			// TODO asserts
			m.SetStatus(Evaluated)
			m.SetEvaluationError(err)
		}
		// TODO asserts
		return err
	}

	// TODO asserts
	return nil
}

func (rt *Runtime) innerModuleEvaluation(m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var c CyclicModuleRecord
	var ok bool
	if c, ok = m.(CyclicModuleRecord); !ok {
		return index, m.Evaluate()
	}
	if status := c.Status(); status == Evaluated { // TODO switch
		return index, c.EvaluationError()
	} else if status == Evaluating {
		return index, nil
	} else if status != Linked {
		return 0, errors.New("module isn't linked when it's being evaluated")
	}
	c.SetStatus(Evaluating)
	c.SetDFSIndex(index)
	c.SetDFSAncestorIndex(index)
	index++

	*stack = append(*stack, c)
	var err error
	var requiredModule ModuleRecord
	for _, required := range c.RequestedModules() {
		requiredModule, err = rt.hostResolveImportedModule(c, required)
		if err != nil {
			return 0, err
		}
		index, err = rt.innerModuleEvaluation(requiredModule, stack, index)
		if err != nil {
			return 0, err
		}
		if requiredC, ok := requiredModule.(CyclicModuleRecord); ok {
			// TODO some asserts
			if requiredC.Status() == Evaluating {
				if ancestorIndex := c.DFSAncestorIndex(); requiredC.DFSAncestorIndex() > ancestorIndex {
					requiredC.SetDFSAncestorIndex(ancestorIndex)
				}
			}
		}
	}
	err = c.ExecuteModule()
	if err != nil {
		return 0, err
	}
	// TODO asserts

	if c.DFSAncestorIndex() == c.DFSIndex() {
		for i := len(*stack) - 1; i >= 0; i-- {
			requiredModule := (*stack)[i]
			// TODO assert
			requiredC := requiredModule.(CyclicModuleRecord)
			requiredC.SetStatus(Evaluated)
			if requiredC == c {
				break
			}
		}
	}
	return index, nil
}

var _ CyclicModuleRecord = &SourceTextModuleRecord{}

type SourceTextModuleRecord struct {
	cyclicModuleStub
	scope    *scope
	rt       *Runtime  // TODO this is not great as it means the whole thing needs to be reparsed for each runtime
	compiler *compiler // TODO remove this
	body     *ast.Program
	// context
	// importmeta
	importEntries         []importEntry
	localExportEntries    []exportEntry
	indirectExportEntries []exportEntry
	starExportEntries     []exportEntry
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
				result = append(result, importEntry{
					moduleRequest: moduleRequest,
					importName:    el.IdentifierName.String(),
					localName:     el.Alias.String(),
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
	// spew.Dump(declarations)
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
				fmt.Printf("unimplemented %+v\n", exportDeclaration.ExportFromClause)
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
			localName := "default"
			if hoistable.FunctionDeclaration.Name != nil {
				localName = string(hoistable.FunctionDeclaration.Name.Name.String())
			}
			result = append(result, exportEntry{
				localName:  localName,
				exportName: "default",
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
				fmt.Printf("unimplemented %+v\n", exportDeclaration.ExportFromClause)
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
			fmt.Printf("unimplemented %+v\n", exportDeclaration)
			panic("wat")

		}
	}
	// spew.Dump(result)
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
func (rt *Runtime) ParseModule(sourceText string) (*SourceTextModuleRecord, error) {
	// TODO asserts
	opts := append(rt.parserOptions, parser.IsModule)
	body, err := Parse("module", sourceText, opts...)
	_ = body
	if err != nil {
		return nil, err
	}
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
	}

	s.rt = rt
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

func (module *SourceTextModuleRecord) ExecuteModule() error {
	// TODO copy runtime.RunProgram here with some changes so that it doesn't touch the global ?

	_, err := module.rt.RunProgram(module.compiler.p)
	return err
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
		requestedModule, err := module.rt.hostResolveImportedModule(module, e.moduleRequest)
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

	// TODO catch panics/exceptions
	module.compiler.compileModule(module)
	return
	/* this is in the compiler
	for _, e := range module.indirectExportEntries {
		resolution := module.ResolveExport(e.exportName)
		if resolution == nil { // TODO or ambiguous
			panic(module.rt.newSyntaxError("bad resolution", -1)) // TODO fix
		}
		// TODO asserts
	}
	for _, in := range module.importEntries {
		importedModule := module.rt.hostResolveImportedModule(module, in.moduleRequest)
		if in.importName == "*" {
			namespace := getModuleNamespace(importedModule)
			b, exists := module.compiler.scope.bindName(in.localName)
			if exists {
				panic("this bad?")
			}
			b.emitInit()
		}

	}

	return nil // TODO implement
	*/
}

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
			importedModule, err := module.rt.hostResolveImportedModule(module, e.moduleRequest)
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
		importedModule, err := module.rt.hostResolveImportedModule(module, e.moduleRequest)
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

func (module *SourceTextModuleRecord) Evaluate() error {
	return module.rt.CyclicModuleRecordEvaluate(module)
}

func (module *SourceTextModuleRecord) Link() error {
	return module.compiler.CyclicModuleRecordConcreteLink(module)
}

type cyclicModuleStub struct {
	namespace        *Namespace
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

func (c *cyclicModuleStub) Namespace() *Namespace {
	return c.namespace
}

func (c *cyclicModuleStub) SetNamespace(namespace *Namespace) {
	c.namespace = namespace
}
