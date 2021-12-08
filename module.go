package goja

import (
	"errors"
	"sort"

	"github.com/dop251/goja/ast"
)

// TODO most things here probably should be unexported and names should be revised before merged in master
// Record should probably be dropped from everywhere

// ModuleRecord is the common interface for module record as defined in the EcmaScript specification
type ModuleRecord interface {
	GetExportedNames(resolveset ...*SourceTextModuleRecord) []string // TODO maybe this parameter is wrong
	ResolveExport(exportName string, resolveset ...string) *Value    // TODO this probably should not return Value directly
	Link() error
	Evaluate() error
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

func (rt *Runtime) CyclicModuleRecordConcreteLink(c CyclicModuleRecord) error {
	if c.Status() == Linking || c.Status() == Evaluating {
		return errors.New("bad status on link")
	}

	stack := []CyclicModuleRecord{}
	if _, err := rt.innerModuleLinking(c, &stack, 0); err != nil {
		for _, m := range stack {
			if m.Status() != Linking {
				return errors.New("bad status on link")
			}
			m.SetStatus(Unlinked)

			// TODO reset the rest

		}
		c.SetStatus(Unlinked)
		return err

	}
	return nil
}

func (rt *Runtime) innerModuleLinking(m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var c CyclicModuleRecord
	var ok bool
	if c, ok = m.(CyclicModuleRecord); !ok {
		return index, m.Link()
	}
	if status := c.Status(); status == Linking || status == Linked || status == Evaluated {
		return index, nil
	} else if status != Unlinked {
		return 0, errors.New("bad status on link") // TODO fix
	}
	c.SetStatus(Linking)
	c.SetDFSIndex(index)
	c.SetDFSAncestorIndex(index)
	index++
	*stack = append(*stack, c)
	var err error
	for _, required := range c.RequestedModules() {
		requiredModule := rt.hostResolveImportedModule(c, required)
		index, err = rt.innerModuleLinking(requiredModule, stack, index)
		if err != nil {
			return 0, err
		}
		if requiredC, ok := requiredModule.(CyclicModuleRecord); ok {
			// TODO some asserts
			if requiredC.Status() == Linking {
				if ancestorIndex := c.DFSAncestorIndex(); requiredC.DFSAncestorIndex() > ancestorIndex {
					requiredC.SetDFSAncestorIndex(ancestorIndex)
				}
			}
		}
	}
	c.InitializeEnvorinment() // TODO implement
	// TODO more asserts

	if c.DFSAncestorIndex() == c.DFSIndex() {
		for {
			requiredModule := (*stack)[len(*stack)-1]
			// TODO assert
			requiredC := requiredModule.(CyclicModuleRecord)
			requiredC.SetStatus(Linked)
			if requiredC == c {
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
	} else if status != Evaluating {
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
	for _, required := range c.RequestedModules() {
		requiredModule := rt.hostResolveImportedModule(c, required)
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
	c.ExecuteModule()
	// TODO asserts

	if c.DFSAncestorIndex() == c.DFSIndex() {
		for {
			requiredModule := (*stack)[len(*stack)-1]
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
	rt   *Runtime // TODO this is not great as it means the whole thing needs to be reparsed for each runtime
	body *ast.Program
	// context
	// importmeta
	// importEntries
	localExportEntries    []exportEntry
	indirectExportEntries []exportEntry
	starExportEntries     []exportEntry
}

type importEntry struct {
	moduleRequest string
	importName    string
	localName     string
}

type exportEntry struct {
	exportName    string
	moduleRequest string
	importName    string
	localName     string
}

func includes(slice []string, s string) bool {
	i := sort.SearchStrings(slice, s)
	return i < len(slice) && slice[i] == s
}

// This should probably be part of Parse
// TODO arguments to this need fixing
func (rt *Runtime) ParseModule(sourceText string) (*SourceTextModuleRecord, error) {
	// TODO asserts
	body, err := Parse("module", sourceText, rt.parserOptions...)
	_ = body
	if err != nil {
		return nil, err
	}
	// Let body be ParseText(sourceText, Module).
	// 3. If body is a List of errors, return body.
	// 4. Let requestedModules be the ModuleRequests of body.
	// 5. Let importEntries be ImportEntries of body.
	// importEntries := body.ImportEntries TODO fix
	// 6. Let importedBoundNames be ImportedLocalNames(importEntries).
	var importedBoundNames []string // fix
	// 7. Let indirectExportEntries be a new empty List.
	// 8. Let localExportEntries be a new empty List.
	var localExportEntries []exportEntry // fix
	// 9. Let starExportEntries be a new empty List.
	// 10. Let exportEntries be ExportEntries of body.
	var exportEntries []exportEntry
	for _, exportDeclarion := range body.ExportEntries {
		for _, spec := range exportDeclarion.ExportFromClause.NamedExports.ExportsList {
			exportEntries = append(exportEntries, exportEntry{
				localName:  spec.IdentifierName.String(),
				exportName: spec.Alias.String(),
			})
		}
	}
	for _, ee := range exportEntries {
		if ee.moduleRequest == "" { // technically nil
			if !includes(importedBoundNames, ee.localName) { // TODO make it not true always
				localExportEntries = append(localExportEntries, ee)
			} else {
				// TODO logic when we reexport something imported
			}
		} else {
			// TODO implement this where we have export {s } from "somewhere"; and co.
		}
	}
	return &SourceTextModuleRecord{
		// realm isn't implement
		// environment is undefined
		// namespace is undefined
		cyclicModuleStub: cyclicModuleStub{
			status: Unlinked,
		},
		// EvaluationError is undefined
		// hostDefined TODO
		body: body,
		// Context empty
		// importmenta empty
		// RequestedModules
		// ImportEntries
		localExportEntries: localExportEntries,
		// indirectExportEntries TODO
		// starExportEntries TODO
		// DFSIndex is empty
		// DFSAncestorIndex is empty
	}, nil // TODO fix
}

func (s *SourceTextModuleRecord) ExecuteModule() error {
	// TODO copy runtime.RunProgram here with some changes so that it doesn't touch the global ?
	return nil
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
		requestedModule := module.rt.hostResolveImportedModule(module, e.moduleRequest)
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

func (s *SourceTextModuleRecord) InitializeEnvorinment() error {
	return nil // TODO implement
}

func (s *SourceTextModuleRecord) ResolveExport(exportname string, resolveset ...string) *Value {
	return nil // TODO implement
}

func (s *SourceTextModuleRecord) Evaluate() error {
	return s.rt.CyclicModuleRecordEvaluate(s)
}

func (s *SourceTextModuleRecord) Link() error {
	return s.rt.CyclicModuleRecordConcreteLink(s)
}

type cyclicModuleStub struct {
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
