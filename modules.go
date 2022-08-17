package goja

import (
	"errors"
)

type HostResolveImportedModuleFunc func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error)

// TODO most things here probably should be unexported and names should be revised before merged in master
// Record should probably be dropped from everywhere

// ModuleRecord is the common interface for module record as defined in the EcmaScript specification
type ModuleRecord interface {
	GetExportedNames(resolveset ...ModuleRecord) []string
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
	Instantiate(rt *Runtime) (CyclicModuleInstance, error)
}

type (
	ModuleInstance interface {
		GetBindingValue(string) Value
	}
	CyclicModuleInstance interface {
		ModuleInstance
		ExecuteModule(*Runtime) (CyclicModuleInstance, error)
	}
)

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

func (c *compiler) CyclicModuleRecordConcreteLink(module ModuleRecord) error {
	stack := []CyclicModuleRecord{}
	if _, err := c.innerModuleLinking(newLinkState(), module, &stack, 0); err != nil {
		return err
	}
	return nil
}

func (c *compiler) innerModuleLinking(state *linkState, m ModuleRecord, stack *[]CyclicModuleRecord, index uint) (uint, error) {
	var module CyclicModuleRecord
	var ok bool
	if module, ok = m.(CyclicModuleRecord); !ok {
		return index, m.Link()
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
		c, err = cr.Instantiate(r)
		if err != nil {
			return nil, index, err
		}

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
	for _, required := range cr.RequestedModules() {
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

func (r *Runtime) GetActiveScriptOrModule() interface{} { // have some better type
	if r.vm.prg != nil && r.vm.prg.scriptOrModule != nil {
		return r.vm.prg.scriptOrModule
	}
	for i := len(r.vm.callStack) - 1; i >= 0; i-- {
		prg := r.vm.callStack[i].prg
		if prg.scriptOrModule != nil {
			return prg.scriptOrModule
		}
	}
	return nil
}

func (r *Runtime) getImportMetaFor(m ModuleRecord) *Object {
	if r.importMetas == nil {
		r.importMetas = make(map[ModuleRecord]*Object)
	}
	if o, ok := r.importMetas[m]; ok {
		return o
	}
	o := r.NewObject()
	o.SetPrototype(nil)

	var properties []MetaProperty
	if r.getImportMetaProperties != nil {
		properties = r.getImportMetaProperties(m)
	}

	for _, property := range properties {
		o.Set(property.Key, property.Value)
	}

	if r.finalizeImportMeta != nil {
		r.finalizeImportMeta(o, m)
	}

	r.importMetas[m] = o
	return o
}

type MetaProperty struct {
	Key   string
	Value Value
}

func (r *Runtime) SetGetImportMetaProperties(fn func(ModuleRecord) []MetaProperty) {
	r.getImportMetaProperties = fn
}

func (r *Runtime) SetFinalImportMeta(fn func(*Object, ModuleRecord)) {
	r.finalizeImportMeta = fn
}

// TODO fix signature
type ImportModuleDynamicallyCallback func(referencingScriptOrModule interface{}, specifier Value, promiseCapability interface{})

func (r *Runtime) SetImportModuleDynamically(callback ImportModuleDynamicallyCallback) {
	r.importModuleDynamically = callback
}

// TODO figure out the arguments
func (r *Runtime) FinalizeDynamicImport(m ModuleRecord, pcap interface{}, err interface{}) {
	p := pcap.(*promiseCapability)
	if err != nil {
		switch x1 := err.(type) {
		case *Exception:
			p.reject(x1.val)
		case *CompilerSyntaxError:
			p.reject(r.builtin_new(r.global.SyntaxError, []Value{newStringValue(x1.Error())}))
		case *CompilerReferenceError:
			p.reject(r.newError(r.global.ReferenceError, x1.Message))
		default:
			p.reject(r.ToValue(err))
		}
		return
	}
	p.resolve(r.NamespaceObjectFor(m))
}
