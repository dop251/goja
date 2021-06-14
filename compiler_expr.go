package goja

import (
	"fmt"
	"regexp"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/token"
	"github.com/dop251/goja/unistring"
)

var (
	octalRegexp = regexp.MustCompile(`^0[0-7]`)
)

type compiledExpr interface {
	emitGetter(putOnStack bool)
	emitSetter(valueExpr compiledExpr, putOnStack bool)
	emitRef()
	emitUnary(prepare, body func(), postfix, putOnStack bool)
	deleteExpr() compiledExpr
	constant() bool
	addSrcMap()
}

type compiledExprOrRef interface {
	compiledExpr
	emitGetterOrRef()
}

type compiledCallExpr struct {
	baseCompiledExpr
	args   []compiledExpr
	callee compiledExpr
}

type compiledObjectLiteral struct {
	baseCompiledExpr
	expr *ast.ObjectLiteral
}

type compiledArrayLiteral struct {
	baseCompiledExpr
	expr *ast.ArrayLiteral
}

type compiledRegexpLiteral struct {
	baseCompiledExpr
	expr *ast.RegExpLiteral
}

type compiledLiteral struct {
	baseCompiledExpr
	val Value
}

type compiledAssignExpr struct {
	baseCompiledExpr
	left, right compiledExpr
	operator    token.Token
}

type compiledObjectAssignmentPattern struct {
	baseCompiledExpr
	expr *ast.ObjectPattern
}

type compiledArrayAssignmentPattern struct {
	baseCompiledExpr
	expr *ast.ArrayPattern
}

type deleteGlobalExpr struct {
	baseCompiledExpr
	name unistring.String
}

type deleteVarExpr struct {
	baseCompiledExpr
	name unistring.String
}

type deletePropExpr struct {
	baseCompiledExpr
	left compiledExpr
	name unistring.String
}

type deleteElemExpr struct {
	baseCompiledExpr
	left, member compiledExpr
}

type constantExpr struct {
	baseCompiledExpr
	val Value
}

type baseCompiledExpr struct {
	c      *compiler
	offset int
}

type compiledIdentifierExpr struct {
	baseCompiledExpr
	name unistring.String
}

type compiledFunctionLiteral struct {
	baseCompiledExpr
	expr    *ast.FunctionLiteral
	lhsName unistring.String
	isExpr  bool
	strict  bool
}

type compiledBracketExpr struct {
	baseCompiledExpr
	left, member compiledExpr
}

type compiledThisExpr struct {
	baseCompiledExpr
}

type compiledNewExpr struct {
	baseCompiledExpr
	callee compiledExpr
	args   []compiledExpr
}

type compiledNewTarget struct {
	baseCompiledExpr
}

type compiledSequenceExpr struct {
	baseCompiledExpr
	sequence []compiledExpr
}

type compiledUnaryExpr struct {
	baseCompiledExpr
	operand  compiledExpr
	operator token.Token
	postfix  bool
}

type compiledConditionalExpr struct {
	baseCompiledExpr
	test, consequent, alternate compiledExpr
}

type compiledLogicalOr struct {
	baseCompiledExpr
	left, right compiledExpr
}

type compiledLogicalAnd struct {
	baseCompiledExpr
	left, right compiledExpr
}

type compiledBinaryExpr struct {
	baseCompiledExpr
	left, right compiledExpr
	operator    token.Token
}

type compiledEnumGetExpr struct {
	baseCompiledExpr
}

type defaultDeleteExpr struct {
	baseCompiledExpr
	expr compiledExpr
}

func (e *defaultDeleteExpr) emitGetter(putOnStack bool) {
	e.expr.emitGetter(false)
	if putOnStack {
		e.c.emit(loadVal(e.c.p.defineLiteralValue(valueTrue)))
	}
}

func (c *compiler) compileExpression(v ast.Expression) compiledExpr {
	// log.Printf("compileExpression: %T", v)
	switch v := v.(type) {
	case nil:
		return nil
	case *ast.AssignExpression:
		return c.compileAssignExpression(v)
	case *ast.NumberLiteral:
		return c.compileNumberLiteral(v)
	case *ast.StringLiteral:
		return c.compileStringLiteral(v)
	case *ast.BooleanLiteral:
		return c.compileBooleanLiteral(v)
	case *ast.NullLiteral:
		r := &compiledLiteral{
			val: _null,
		}
		r.init(c, v.Idx0())
		return r
	case *ast.Identifier:
		return c.compileIdentifierExpression(v)
	case *ast.CallExpression:
		return c.compileCallExpression(v)
	case *ast.ObjectLiteral:
		return c.compileObjectLiteral(v)
	case *ast.ArrayLiteral:
		return c.compileArrayLiteral(v)
	case *ast.RegExpLiteral:
		return c.compileRegexpLiteral(v)
	case *ast.BinaryExpression:
		return c.compileBinaryExpression(v)
	case *ast.UnaryExpression:
		return c.compileUnaryExpression(v)
	case *ast.ConditionalExpression:
		return c.compileConditionalExpression(v)
	case *ast.FunctionLiteral:
		return c.compileFunctionLiteral(v, true)
	case *ast.DotExpression:
		return c.compileDotExpr(v)
	case *ast.BracketExpression:
		return c.compileBracketExpr(v)
	case *ast.ThisExpression:
		r := &compiledThisExpr{}
		r.init(c, v.Idx0())
		return r
	case *ast.SequenceExpression:
		return c.compileSequenceExpression(v)
	case *ast.NewExpression:
		return c.compileNewExpression(v)
	case *ast.MetaProperty:
		return c.compileMetaProperty(v)
	case *ast.ObjectPattern:
		return c.compileObjectAssignmentPattern(v)
	case *ast.ArrayPattern:
		return c.compileArrayAssignmentPattern(v)
	default:
		panic(fmt.Errorf("Unknown expression type: %T", v))
	}
}

func (e *baseCompiledExpr) constant() bool {
	return false
}

func (e *baseCompiledExpr) init(c *compiler, idx file.Idx) {
	e.c = c
	e.offset = int(idx) - 1
}

func (e *baseCompiledExpr) emitSetter(compiledExpr, bool) {
	e.c.throwSyntaxError(e.offset, "Not a valid left-value expression")
}

func (e *baseCompiledExpr) emitRef() {
	e.c.throwSyntaxError(e.offset, "Cannot emit reference for this type of expression")
}

func (e *baseCompiledExpr) deleteExpr() compiledExpr {
	r := &constantExpr{
		val: valueTrue,
	}
	r.init(e.c, file.Idx(e.offset+1))
	return r
}

func (e *baseCompiledExpr) emitUnary(func(), func(), bool, bool) {
	e.c.throwSyntaxError(e.offset, "Not a valid left-value expression")
}

func (e *baseCompiledExpr) addSrcMap() {
	if e.offset > 0 {
		e.c.p.srcMap = append(e.c.p.srcMap, srcMapItem{pc: len(e.c.p.code), srcPos: e.offset})
	}
}

func (e *constantExpr) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		e.c.emit(loadVal(e.c.p.defineLiteralValue(e.val)))
	}
}

func (e *compiledIdentifierExpr) emitGetter(putOnStack bool) {
	e.addSrcMap()
	if b, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if b != nil {
			if putOnStack {
				b.emitGet()
			} else {
				b.emitGetP()
			}
		} else {
			panic("No dynamics and not found")
		}
	} else {
		if b != nil {
			b.emitGetVar(false)
		} else {
			e.c.emit(loadDynamic(e.name))
		}
		if !putOnStack {
			e.c.emit(pop)
		}
	}
}

func (e *compiledIdentifierExpr) emitGetterOrRef() {
	e.addSrcMap()
	if b, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if b != nil {
			b.emitGet()
		} else {
			panic("No dynamics and not found")
		}
	} else {
		if b != nil {
			b.emitGetVar(false)
		} else {
			e.c.emit(loadDynamicRef(e.name))
		}
	}
}

func (e *compiledIdentifierExpr) emitGetterAndCallee() {
	e.addSrcMap()
	if b, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if b != nil {
			e.c.emit(loadUndef)
			b.emitGet()
		} else {
			panic("No dynamics and not found")
		}
	} else {
		if b != nil {
			b.emitGetVar(true)
		} else {
			e.c.emit(loadDynamicCallee(e.name))
		}
	}
}

func (c *compiler) emitVarSetter1(name unistring.String, offset int, putOnStack bool, emitRight func(isRef bool)) {
	if c.scope.strict {
		c.checkIdentifierLName(name, offset)
	}

	if b, noDynamics := c.scope.lookupName(name); noDynamics {
		emitRight(false)
		if b != nil {
			if putOnStack {
				b.emitSet()
			} else {
				b.emitSetP()
			}
		} else {
			if c.scope.strict {
				c.emit(setGlobalStrict(name))
			} else {
				c.emit(setGlobal(name))
			}
			if !putOnStack {
				c.emit(pop)
			}
		}
	} else {
		if b != nil {
			b.emitResolveVar(c.scope.strict)
		} else {
			if c.scope.strict {
				c.emit(resolveVar1Strict(name))
			} else {
				c.emit(resolveVar1(name))
			}
		}
		emitRight(true)
		if putOnStack {
			c.emit(putValue)
		} else {
			c.emit(putValueP)
		}
	}
}

func (c *compiler) emitVarSetter(name unistring.String, offset int, valueExpr compiledExpr, putOnStack bool) {
	c.emitVarSetter1(name, offset, putOnStack, func(bool) {
		c.emitExpr(valueExpr, true)
	})
}

func (c *compiler) emitVarRef(name unistring.String, offset int) {
	if c.scope.strict {
		c.checkIdentifierLName(name, offset)
	}

	b, _ := c.scope.lookupName(name)
	if b != nil {
		b.emitResolveVar(c.scope.strict)
	} else {
		if c.scope.strict {
			c.emit(resolveVar1Strict(name))
		} else {
			c.emit(resolveVar1(name))
		}
	}
}

func (e *compiledIdentifierExpr) emitRef() {
	e.c.emitVarRef(e.name, e.offset)
}

func (e *compiledIdentifierExpr) emitSetter(valueExpr compiledExpr, putOnStack bool) {
	e.c.emitVarSetter(e.name, e.offset, valueExpr, putOnStack)
}

func (e *compiledIdentifierExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if putOnStack {
		e.c.emitVarSetter1(e.name, e.offset, true, func(isRef bool) {
			e.c.emit(loadUndef)
			if isRef {
				e.c.emit(getValue)
			} else {
				e.emitGetter(true)
			}
			if prepare != nil {
				prepare()
			}
			if !postfix {
				body()
			}
			e.c.emit(rdupN(1))
			if postfix {
				body()
			}
		})
		e.c.emit(pop)
	} else {
		e.c.emitVarSetter1(e.name, e.offset, false, func(isRef bool) {
			if isRef {
				e.c.emit(getValue)
			} else {
				e.emitGetter(true)
			}
			body()
		})
	}
}

func (e *compiledIdentifierExpr) deleteExpr() compiledExpr {
	if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		panic("Unreachable")
	}
	if b, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if b == nil {
			r := &deleteGlobalExpr{
				name: e.name,
			}
			r.init(e.c, file.Idx(0))
			return r
		}
	} else {
		if b == nil {
			r := &deleteVarExpr{
				name: e.name,
			}
			r.init(e.c, file.Idx(e.offset+1))
			return r
		}
	}
	r := &compiledLiteral{
		val: valueFalse,
	}
	r.init(e.c, file.Idx(e.offset+1))
	return r
}

type compiledDotExpr struct {
	baseCompiledExpr
	left compiledExpr
	name unistring.String
}

func (e *compiledDotExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.addSrcMap()
	e.c.emit(getProp(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledDotExpr) emitRef() {
	e.left.emitGetter(true)
	if e.c.scope.strict {
		e.c.emit(getPropRefStrict(e.name))
	} else {
		e.c.emit(getPropRef(e.name))
	}
}

func (e *compiledDotExpr) emitSetter(valueExpr compiledExpr, putOnStack bool) {
	e.left.emitGetter(true)
	valueExpr.emitGetter(true)
	if e.c.scope.strict {
		if putOnStack {
			e.c.emit(setPropStrict(e.name))
		} else {
			e.c.emit(setPropStrictP(e.name))
		}
	} else {
		if putOnStack {
			e.c.emit(setProp(e.name))
		} else {
			e.c.emit(setPropP(e.name))
		}
	}
}

func (e *compiledDotExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if !putOnStack {
		e.left.emitGetter(true)
		e.c.emit(dup)
		e.c.emit(getProp(e.name))
		body()
		if e.c.scope.strict {
			e.c.emit(setPropStrict(e.name), pop)
		} else {
			e.c.emit(setProp(e.name), pop)
		}
	} else {
		if !postfix {
			e.left.emitGetter(true)
			e.c.emit(dup)
			e.c.emit(getProp(e.name))
			if prepare != nil {
				prepare()
			}
			body()
			if e.c.scope.strict {
				e.c.emit(setPropStrict(e.name))
			} else {
				e.c.emit(setProp(e.name))
			}
		} else {
			e.c.emit(loadUndef)
			e.left.emitGetter(true)
			e.c.emit(dup)
			e.c.emit(getProp(e.name))
			if prepare != nil {
				prepare()
			}
			e.c.emit(rdupN(2))
			body()
			if e.c.scope.strict {
				e.c.emit(setPropStrict(e.name))
			} else {
				e.c.emit(setProp(e.name))
			}
			e.c.emit(pop)
		}
	}
}

func (e *compiledDotExpr) deleteExpr() compiledExpr {
	r := &deletePropExpr{
		left: e.left,
		name: e.name,
	}
	r.init(e.c, file.Idx(0))
	return r
}

func (e *compiledBracketExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	e.addSrcMap()
	e.c.emit(getElem)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledBracketExpr) emitRef() {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	if e.c.scope.strict {
		e.c.emit(getElemRefStrict)
	} else {
		e.c.emit(getElemRef)
	}
}

func (e *compiledBracketExpr) emitSetter(valueExpr compiledExpr, putOnStack bool) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	valueExpr.emitGetter(true)
	if e.c.scope.strict {
		if putOnStack {
			e.c.emit(setElemStrict)
		} else {
			e.c.emit(setElemStrictP)
		}
	} else {
		if putOnStack {
			e.c.emit(setElem)
		} else {
			e.c.emit(setElemP)
		}
	}
}

func (e *compiledBracketExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if !putOnStack {
		e.left.emitGetter(true)
		e.member.emitGetter(true)
		e.c.emit(dupN(1), dupN(1))
		e.c.emit(getElem)
		body()
		if e.c.scope.strict {
			e.c.emit(setElemStrict, pop)
		} else {
			e.c.emit(setElem, pop)
		}
	} else {
		if !postfix {
			e.left.emitGetter(true)
			e.member.emitGetter(true)
			e.c.emit(dupN(1), dupN(1))
			e.c.emit(getElem)
			if prepare != nil {
				prepare()
			}
			body()
			if e.c.scope.strict {
				e.c.emit(setElemStrict)
			} else {
				e.c.emit(setElem)
			}
		} else {
			e.c.emit(loadUndef)
			e.left.emitGetter(true)
			e.member.emitGetter(true)
			e.c.emit(dupN(1), dupN(1))
			e.c.emit(getElem)
			if prepare != nil {
				prepare()
			}
			e.c.emit(rdupN(3))
			body()
			if e.c.scope.strict {
				e.c.emit(setElemStrict, pop)
			} else {
				e.c.emit(setElem, pop)
			}
		}
	}
}

func (e *compiledBracketExpr) deleteExpr() compiledExpr {
	r := &deleteElemExpr{
		left:   e.left,
		member: e.member,
	}
	r.init(e.c, file.Idx(0))
	return r
}

func (e *deleteElemExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	e.addSrcMap()
	if e.c.scope.strict {
		e.c.emit(deleteElemStrict)
	} else {
		e.c.emit(deleteElem)
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deletePropExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.addSrcMap()
	if e.c.scope.strict {
		e.c.emit(deletePropStrict(e.name))
	} else {
		e.c.emit(deleteProp(e.name))
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deleteVarExpr) emitGetter(putOnStack bool) {
	/*if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		return
	}*/
	e.c.emit(deleteVar(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deleteGlobalExpr) emitGetter(putOnStack bool) {
	/*if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		return
	}*/

	e.c.emit(deleteGlobal(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledAssignExpr) emitGetter(putOnStack bool) {
	e.addSrcMap()
	switch e.operator {
	case token.ASSIGN:
		if fn, ok := e.right.(*compiledFunctionLiteral); ok {
			if fn.expr.Name == nil {
				if id, ok := e.left.(*compiledIdentifierExpr); ok {
					fn.lhsName = id.name
				}
			}
		}
		e.left.emitSetter(e.right, putOnStack)
	case token.PLUS:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(add)
		}, false, putOnStack)
	case token.MINUS:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sub)
		}, false, putOnStack)
	case token.MULTIPLY:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(mul)
		}, false, putOnStack)
	case token.SLASH:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(div)
		}, false, putOnStack)
	case token.REMAINDER:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(mod)
		}, false, putOnStack)
	case token.OR:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(or)
		}, false, putOnStack)
	case token.AND:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(and)
		}, false, putOnStack)
	case token.EXCLUSIVE_OR:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(xor)
		}, false, putOnStack)
	case token.SHIFT_LEFT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sal)
		}, false, putOnStack)
	case token.SHIFT_RIGHT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sar)
		}, false, putOnStack)
	case token.UNSIGNED_SHIFT_RIGHT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(shr)
		}, false, putOnStack)
	default:
		panic(fmt.Errorf("Unknown assign operator: %s", e.operator.String()))
	}
}

func (e *compiledLiteral) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		e.c.emit(loadVal(e.c.p.defineLiteralValue(e.val)))
	}
}

func (e *compiledLiteral) constant() bool {
	return true
}

func (c *compiler) compileParameterBindingIdentifier(name unistring.String, offset int) (*binding, bool) {
	if c.scope.strict {
		c.checkIdentifierName(name, offset)
		c.checkIdentifierLName(name, offset)
	}
	b, unique := c.scope.bindNameShadow(name)
	if !unique && c.scope.strict {
		c.throwSyntaxError(offset, "Strict mode function may not have duplicate parameter names (%s)", name)
		return nil, false
	}
	return b, unique
}

func (c *compiler) compileParameterPatternIdBinding(name unistring.String, offset int) {
	if _, unique := c.compileParameterBindingIdentifier(name, offset); !unique {
		c.throwSyntaxError(offset, "Duplicate parameter name not allowed in this context")
	}
}

func (c *compiler) compileParameterPatternBinding(item ast.Expression) {
	c.createBindings(item, c.compileParameterPatternIdBinding)
}

func (e *compiledFunctionLiteral) emitGetter(putOnStack bool) {
	savedPrg := e.c.p
	e.c.p = &Program{
		src: e.c.p.src,
	}
	e.c.newScope()
	e.c.scope.function = true

	var name unistring.String
	if e.expr.Name != nil {
		name = e.expr.Name.Name
	} else {
		name = e.lhsName
	}

	if name != "" {
		e.c.p.funcName = name
	}
	savedBlock := e.c.block
	defer func() {
		e.c.block = savedBlock
	}()

	e.c.block = &block{
		typ: blockScope,
	}

	if !e.c.scope.strict {
		e.c.scope.strict = e.strict
	}

	hasPatterns := false
	hasInits := false
	firstDupIdx := -1

	if e.expr.ParameterList.Rest != nil {
		hasPatterns = true // strictly speaking not, but we need to activate all the checks
	}

	// First, make sure that the first bindings correspond to the formal parameters
	for _, item := range e.expr.ParameterList.List {
		switch tgt := item.Target.(type) {
		case *ast.Identifier:
			offset := int(tgt.Idx) - 1
			b, unique := e.c.compileParameterBindingIdentifier(tgt.Name, offset)
			if !unique {
				firstDupIdx = offset
			}
			b.isArg = true
			b.isVar = true
		case ast.Pattern:
			e.c.scope.addBinding(int(item.Idx0()) - 1)
			hasPatterns = true
		default:
			e.c.throwSyntaxError(int(item.Idx0())-1, "Unsupported BindingElement type: %T", item)
			return
		}
		if item.Initializer != nil {
			hasInits = true
		}
		if (hasPatterns || hasInits) && firstDupIdx >= 0 {
			e.c.throwSyntaxError(firstDupIdx, "Duplicate parameter name not allowed in this context")
			return
		}
	}

	// create pattern bindings
	if hasPatterns {
		for _, item := range e.expr.ParameterList.List {
			switch tgt := item.Target.(type) {
			case *ast.Identifier:
				// we already created those in the previous loop, skipping
			default:
				e.c.compileParameterPatternBinding(tgt)
			}
		}
		if rest := e.expr.ParameterList.Rest; rest != nil {
			e.c.compileParameterPatternBinding(rest)
		}
	}

	paramsCount := len(e.expr.ParameterList.List)

	e.c.scope.numArgs = paramsCount
	e.c.compileDeclList(e.expr.DeclarationList, true)
	body := e.expr.Body.List
	funcs := e.c.extractFunctions(body)
	e.c.createFunctionBindings(funcs)
	s := e.c.scope
	e.c.compileLexicalDeclarations(body, true)
	var calleeBinding *binding
	if e.isExpr && e.expr.Name != nil {
		if b, created := s.bindName(e.expr.Name.Name); created {
			calleeBinding = b
		}
	}
	preambleLen := 4 // enter, boxThis, createArgs, set
	e.c.p.code = make([]instruction, preambleLen, 8)

	emitArgsRestMark := -1

	if hasPatterns || hasInits {
		for i, item := range e.expr.ParameterList.List {
			if pattern, ok := item.Target.(ast.Pattern); ok {
				i := i
				e.c.compilePatternInitExpr(func() {
					e.c.scope.bindings[i].emitGet()
				}, item.Initializer, item.Target.Idx0()).emitGetter(true)
				e.c.emitPattern(pattern, func(target, init compiledExpr) {
					e.c.emitPatternLexicalAssign(target, init, true)
				}, false)
			} else if item.Initializer != nil {
				e.c.scope.bindings[i].emitGet()
				mark := len(e.c.p.code)
				e.c.emit(nil)
				e.c.compileExpression(item.Initializer).emitGetter(true)
				e.c.scope.bindings[i].emitSet()
				e.c.p.code[mark] = jdef(len(e.c.p.code) - mark)
				e.c.emit(pop)
			}
		}
		if rest := e.expr.ParameterList.Rest; rest != nil {
			e.c.emitAssign(rest, &compiledEmitterExpr{
				emitter: func() {
					emitArgsRestMark = len(e.c.p.code)
					e.c.emit(createArgsRestStack(paramsCount))
				},
			}, func(target, init compiledExpr) {
				e.c.emitPatternLexicalAssign(target, init, true)
			})
		}
	}

	if calleeBinding != nil {
		e.c.emit(loadCallee)
		calleeBinding.emitSetP()
	}

	e.c.compileFunctions(funcs)
	e.c.compileStatements(body, false)

	var last ast.Statement
	if l := len(body); l > 0 {
		last = body[l-1]
	}
	if _, ok := last.(*ast.ReturnStatement); !ok {
		e.c.emit(loadUndef, ret)
	}

	delta := 0
	code := e.c.p.code

	if calleeBinding != nil && !s.isDynamic() && calleeBinding.useCount() == 1 {
		s.deleteBinding(calleeBinding)
		preambleLen += 2
	}

	if (s.argsNeeded || s.isDynamic()) && !s.argsInStash {
		s.moveArgsToStash()
	}

	if s.argsNeeded {
		pos := preambleLen - 2
		delta += 2
		if s.strict {
			code[pos] = createArgsUnmapped(paramsCount)
		} else {
			code[pos] = createArgsMapped(paramsCount)
		}
		pos++
		b, _ := s.bindName("arguments")
		e.c.p.code = code[:pos]
		b.emitSetP()
		e.c.p.code = code
	}

	stashSize, stackSize := s.finaliseVarAlloc(0)

	if !s.strict && s.thisNeeded {
		delta++
		code[preambleLen-delta] = boxThis
	}
	delta++
	delta = preambleLen - delta
	var enter instruction
	if stashSize > 0 || s.argsInStash {
		enter1 := enterFunc{
			numArgs:     uint32(paramsCount),
			argsToStash: s.argsInStash,
			stashSize:   uint32(stashSize),
			stackSize:   uint32(stackSize),
			extensible:  s.dynamic,
		}
		if s.isDynamic() {
			enter1.names = s.makeNamesMap()
		}
		enter = &enter1
		if emitArgsRestMark != -1 {
			e.c.p.code[emitArgsRestMark] = createArgsRestStash
		}
	} else {
		enter = &enterFuncStashless{
			stackSize: uint32(stackSize),
			args:      uint32(paramsCount),
		}
	}
	code[delta] = enter
	if delta != 0 {
		e.c.p.code = code[delta:]
		for i := range e.c.p.srcMap {
			e.c.p.srcMap[i].pc -= delta
		}
		s.adjustBase(-delta)
	}

	strict := s.strict
	p := e.c.p
	// e.c.p.dumpCode()
	e.c.popScope()
	e.c.p = savedPrg
	e.c.emit(&newFunc{prg: p, length: uint32(paramsCount), name: name, srcStart: uint32(e.expr.Idx0() - 1), srcEnd: uint32(e.expr.Idx1() - 1), strict: strict})
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileFunctionLiteral(v *ast.FunctionLiteral, isExpr bool) *compiledFunctionLiteral {
	strict := c.scope.strict || c.isStrictStatement(v.Body)
	if v.Name != nil && strict {
		c.checkIdentifierLName(v.Name.Name, int(v.Name.Idx)-1)
	}
	r := &compiledFunctionLiteral{
		expr:   v,
		isExpr: isExpr,
		strict: strict,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledThisExpr) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		scope := e.c.scope
		for ; scope != nil && !scope.function && !scope.eval; scope = scope.outer {
		}

		if scope != nil {
			scope.thisNeeded = true
			e.c.emit(loadStack(0))
		} else {
			e.c.emit(loadGlobalObject)
		}
	}
}

func (e *compiledNewExpr) emitGetter(putOnStack bool) {
	e.callee.emitGetter(true)
	for _, expr := range e.args {
		expr.emitGetter(true)
	}
	e.addSrcMap()
	e.c.emit(_new(len(e.args)))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileNewExpression(v *ast.NewExpression) compiledExpr {
	args := make([]compiledExpr, len(v.ArgumentList))
	for i, expr := range v.ArgumentList {
		args[i] = c.compileExpression(expr)
	}
	r := &compiledNewExpr{
		callee: c.compileExpression(v.Callee),
		args:   args,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledNewTarget) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		e.c.emit(loadNewTarget)
	}
}

func (c *compiler) compileMetaProperty(v *ast.MetaProperty) compiledExpr {
	if v.Meta.Name == "new" || v.Property.Name != "target" {
		r := &compiledNewTarget{}
		r.init(c, v.Idx0())
		return r
	}
	c.throwSyntaxError(int(v.Idx)-1, "Unsupported meta property: %s.%s", v.Meta.Name, v.Property.Name)
	return nil
}

func (e *compiledSequenceExpr) emitGetter(putOnStack bool) {
	if len(e.sequence) > 0 {
		for i := 0; i < len(e.sequence)-1; i++ {
			e.sequence[i].emitGetter(false)
		}
		e.sequence[len(e.sequence)-1].emitGetter(putOnStack)
	}
}

func (c *compiler) compileSequenceExpression(v *ast.SequenceExpression) compiledExpr {
	s := make([]compiledExpr, len(v.Sequence))
	for i, expr := range v.Sequence {
		s[i] = c.compileExpression(expr)
	}
	r := &compiledSequenceExpr{
		sequence: s,
	}
	var idx file.Idx
	if len(v.Sequence) > 0 {
		idx = v.Idx0()
	}
	r.init(c, idx)
	return r
}

func (c *compiler) emitThrow(v Value) {
	if o, ok := v.(*Object); ok {
		t := nilSafe(o.self.getStr("name", nil)).toString().String()
		switch t {
		case "TypeError":
			c.emit(loadDynamic(t))
			msg := o.self.getStr("message", nil)
			if msg != nil {
				c.emit(loadVal(c.p.defineLiteralValue(msg)))
				c.emit(_new(1))
			} else {
				c.emit(_new(0))
			}
			c.emit(throw)
			return
		}
	}
	panic(fmt.Errorf("unknown exception type thrown while evaliating constant expression: %s", v.String()))
}

func (c *compiler) emitConst(expr compiledExpr, putOnStack bool) {
	v, ex := c.evalConst(expr)
	if ex == nil {
		if putOnStack {
			c.emit(loadVal(c.p.defineLiteralValue(v)))
		}
	} else {
		c.emitThrow(ex.val)
	}
}

func (c *compiler) emitExpr(expr compiledExpr, putOnStack bool) {
	if expr.constant() {
		c.emitConst(expr, putOnStack)
	} else {
		expr.emitGetter(putOnStack)
	}
}

func (c *compiler) evalConst(expr compiledExpr) (Value, *Exception) {
	if expr, ok := expr.(*compiledLiteral); ok {
		return expr.val, nil
	}
	if c.evalVM == nil {
		c.evalVM = New().vm
	}
	var savedPrg *Program
	createdPrg := false
	if c.evalVM.prg == nil {
		c.evalVM.prg = &Program{}
		savedPrg = c.p
		c.p = c.evalVM.prg
		createdPrg = true
	}
	savedPc := len(c.p.code)
	expr.emitGetter(true)
	c.emit(halt)
	c.evalVM.pc = savedPc
	ex := c.evalVM.runTry()
	if createdPrg {
		c.evalVM.prg = nil
		c.evalVM.pc = 0
		c.p = savedPrg
	} else {
		c.evalVM.prg.code = c.evalVM.prg.code[:savedPc]
		c.p.code = c.evalVM.prg.code
	}
	if ex == nil {
		return c.evalVM.pop(), nil
	}
	return nil, ex
}

func (e *compiledUnaryExpr) constant() bool {
	return e.operand.constant()
}

func (e *compiledUnaryExpr) emitGetter(putOnStack bool) {
	var prepare, body func()

	toNumber := func() {
		e.c.emit(toNumber)
	}

	switch e.operator {
	case token.NOT:
		e.operand.emitGetter(true)
		e.c.emit(not)
		goto end
	case token.BITWISE_NOT:
		e.operand.emitGetter(true)
		e.c.emit(bnot)
		goto end
	case token.TYPEOF:
		if o, ok := e.operand.(compiledExprOrRef); ok {
			o.emitGetterOrRef()
		} else {
			e.operand.emitGetter(true)
		}
		e.c.emit(typeof)
		goto end
	case token.DELETE:
		e.operand.deleteExpr().emitGetter(putOnStack)
		return
	case token.MINUS:
		e.c.emitExpr(e.operand, true)
		e.c.emit(neg)
		goto end
	case token.PLUS:
		e.c.emitExpr(e.operand, true)
		e.c.emit(plus)
		goto end
	case token.INCREMENT:
		prepare = toNumber
		body = func() {
			e.c.emit(inc)
		}
	case token.DECREMENT:
		prepare = toNumber
		body = func() {
			e.c.emit(dec)
		}
	case token.VOID:
		e.c.emitExpr(e.operand, false)
		if putOnStack {
			e.c.emit(loadUndef)
		}
		return
	default:
		panic(fmt.Errorf("Unknown unary operator: %s", e.operator.String()))
	}

	e.operand.emitUnary(prepare, body, e.postfix, putOnStack)
	return

end:
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileUnaryExpression(v *ast.UnaryExpression) compiledExpr {
	r := &compiledUnaryExpr{
		operand:  c.compileExpression(v.Operand),
		operator: v.Operator,
		postfix:  v.Postfix,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledConditionalExpr) emitGetter(putOnStack bool) {
	e.test.emitGetter(true)
	j := len(e.c.p.code)
	e.c.emit(nil)
	e.consequent.emitGetter(putOnStack)
	j1 := len(e.c.p.code)
	e.c.emit(nil)
	e.c.p.code[j] = jne(len(e.c.p.code) - j)
	e.alternate.emitGetter(putOnStack)
	e.c.p.code[j1] = jump(len(e.c.p.code) - j1)
}

func (c *compiler) compileConditionalExpression(v *ast.ConditionalExpression) compiledExpr {
	r := &compiledConditionalExpr{
		test:       c.compileExpression(v.Test),
		consequent: c.compileExpression(v.Consequent),
		alternate:  c.compileExpression(v.Alternate),
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledLogicalOr) constant() bool {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if v.ToBoolean() {
				return true
			}
			return e.right.constant()
		} else {
			return true
		}
	}

	return false
}

func (e *compiledLogicalOr) emitGetter(putOnStack bool) {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				e.c.emitExpr(e.right, putOnStack)
			} else {
				if putOnStack {
					e.c.emit(loadVal(e.c.p.defineLiteralValue(v)))
				}
			}
		} else {
			e.c.emitThrow(ex.val)
		}
		return
	}
	e.c.emitExpr(e.left, true)
	j := len(e.c.p.code)
	e.addSrcMap()
	e.c.emit(nil)
	e.c.emit(pop)
	e.c.emitExpr(e.right, true)
	e.c.p.code[j] = jeq1(len(e.c.p.code) - j)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledLogicalAnd) constant() bool {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				return true
			} else {
				return e.right.constant()
			}
		} else {
			return true
		}
	}

	return false
}

func (e *compiledLogicalAnd) emitGetter(putOnStack bool) {
	var j int
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				e.c.emit(loadVal(e.c.p.defineLiteralValue(v)))
			} else {
				e.c.emitExpr(e.right, putOnStack)
			}
		} else {
			e.c.emitThrow(ex.val)
		}
		return
	}
	e.left.emitGetter(true)
	j = len(e.c.p.code)
	e.addSrcMap()
	e.c.emit(nil)
	e.c.emit(pop)
	e.c.emitExpr(e.right, true)
	e.c.p.code[j] = jneq1(len(e.c.p.code) - j)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledBinaryExpr) constant() bool {
	return e.left.constant() && e.right.constant()
}

func (e *compiledBinaryExpr) emitGetter(putOnStack bool) {
	e.c.emitExpr(e.left, true)
	e.c.emitExpr(e.right, true)
	e.addSrcMap()

	switch e.operator {
	case token.LESS:
		e.c.emit(op_lt)
	case token.GREATER:
		e.c.emit(op_gt)
	case token.LESS_OR_EQUAL:
		e.c.emit(op_lte)
	case token.GREATER_OR_EQUAL:
		e.c.emit(op_gte)
	case token.EQUAL:
		e.c.emit(op_eq)
	case token.NOT_EQUAL:
		e.c.emit(op_neq)
	case token.STRICT_EQUAL:
		e.c.emit(op_strict_eq)
	case token.STRICT_NOT_EQUAL:
		e.c.emit(op_strict_neq)
	case token.PLUS:
		e.c.emit(add)
	case token.MINUS:
		e.c.emit(sub)
	case token.MULTIPLY:
		e.c.emit(mul)
	case token.SLASH:
		e.c.emit(div)
	case token.REMAINDER:
		e.c.emit(mod)
	case token.AND:
		e.c.emit(and)
	case token.OR:
		e.c.emit(or)
	case token.EXCLUSIVE_OR:
		e.c.emit(xor)
	case token.INSTANCEOF:
		e.c.emit(op_instanceof)
	case token.IN:
		e.c.emit(op_in)
	case token.SHIFT_LEFT:
		e.c.emit(sal)
	case token.SHIFT_RIGHT:
		e.c.emit(sar)
	case token.UNSIGNED_SHIFT_RIGHT:
		e.c.emit(shr)
	default:
		panic(fmt.Errorf("Unknown operator: %s", e.operator.String()))
	}

	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileBinaryExpression(v *ast.BinaryExpression) compiledExpr {

	switch v.Operator {
	case token.LOGICAL_OR:
		return c.compileLogicalOr(v.Left, v.Right, v.Idx0())
	case token.LOGICAL_AND:
		return c.compileLogicalAnd(v.Left, v.Right, v.Idx0())
	}

	r := &compiledBinaryExpr{
		left:     c.compileExpression(v.Left),
		right:    c.compileExpression(v.Right),
		operator: v.Operator,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileLogicalOr(left, right ast.Expression, idx file.Idx) compiledExpr {
	r := &compiledLogicalOr{
		left:  c.compileExpression(left),
		right: c.compileExpression(right),
	}
	r.init(c, idx)
	return r
}

func (c *compiler) compileLogicalAnd(left, right ast.Expression, idx file.Idx) compiledExpr {
	r := &compiledLogicalAnd{
		left:  c.compileExpression(left),
		right: c.compileExpression(right),
	}
	r.init(c, idx)
	return r
}

func (e *compiledObjectLiteral) emitGetter(putOnStack bool) {
	e.addSrcMap()
	e.c.emit(newObject)
	for _, prop := range e.expr.Value {
		switch prop := prop.(type) {
		case *ast.PropertyKeyed:
			keyExpr := e.c.compileExpression(prop.Key)
			computed := false
			var key unistring.String
			switch keyExpr := keyExpr.(type) {
			case *compiledLiteral:
				key = keyExpr.val.string()
			default:
				keyExpr.emitGetter(true)
				computed = true
				//e.c.throwSyntaxError(e.offset, "non-literal properties in object literal are not supported yet")
			}
			valueExpr := e.c.compileExpression(prop.Value)
			var anonFn *compiledFunctionLiteral
			if fn, ok := valueExpr.(*compiledFunctionLiteral); ok {
				if fn.expr.Name == nil {
					anonFn = fn
					fn.lhsName = key
				}
			}
			if computed {
				valueExpr.emitGetter(true)
				switch prop.Kind {
				case ast.PropertyKindValue, ast.PropertyKindMethod:
					if anonFn != nil {
						e.c.emit(setElem1Named)
					} else {
						e.c.emit(setElem1)
					}
				case ast.PropertyKindGet:
					e.c.emit(setPropGetter1)
				case ast.PropertyKindSet:
					e.c.emit(setPropSetter1)
				default:
					panic(fmt.Errorf("unknown property kind: %s", prop.Kind))
				}
			} else {
				if anonFn != nil {
					anonFn.lhsName = key
				}
				valueExpr.emitGetter(true)
				switch prop.Kind {
				case ast.PropertyKindValue:
					if key == __proto__ {
						e.c.emit(setProto)
					} else {
						e.c.emit(setProp1(key))
					}
				case ast.PropertyKindMethod:
					e.c.emit(setProp1(key))
				case ast.PropertyKindGet:
					e.c.emit(setPropGetter(key))
				case ast.PropertyKindSet:
					e.c.emit(setPropSetter(key))
				default:
					panic(fmt.Errorf("unknown property kind: %s", prop.Kind))
				}
			}
		case *ast.PropertyShort:
			key := prop.Name.Name
			if e.c.scope.strict && key == "let" {
				e.c.throwSyntaxError(e.offset, "'let' cannot be used as a shorthand property in strict mode")
			}
			e.c.compileIdentifierExpression(&prop.Name).emitGetter(true)
			e.c.emit(setProp1(key))
		case *ast.SpreadElement:
			e.c.compileExpression(prop.Expression).emitGetter(true)
			e.c.emit(copySpread)
		default:
			panic(fmt.Errorf("unknown Property type: %T", prop))
		}
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileObjectLiteral(v *ast.ObjectLiteral) compiledExpr {
	r := &compiledObjectLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledArrayLiteral) emitGetter(putOnStack bool) {
	e.addSrcMap()
	hasSpread := false
	mark := len(e.c.p.code)
	e.c.emit(nil)
	for _, v := range e.expr.Value {
		if spread, ok := v.(*ast.SpreadElement); ok {
			hasSpread = true
			e.c.compileExpression(spread.Expression).emitGetter(true)
			e.c.emit(pushArraySpread)
		} else {
			if v != nil {
				e.c.compileExpression(v).emitGetter(true)
			} else {
				e.c.emit(loadNil)
			}
			e.c.emit(pushArrayItem)
		}
	}
	var objCount uint32
	if !hasSpread {
		objCount = uint32(len(e.expr.Value))
	}
	e.c.p.code[mark] = newArray(objCount)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileArrayLiteral(v *ast.ArrayLiteral) compiledExpr {
	r := &compiledArrayLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledRegexpLiteral) emitGetter(putOnStack bool) {
	if putOnStack {
		pattern, err := compileRegexp(e.expr.Pattern, e.expr.Flags)
		if err != nil {
			e.c.throwSyntaxError(e.offset, err.Error())
		}

		e.c.emit(&newRegexp{pattern: pattern, src: newStringValue(e.expr.Pattern)})
	}
}

func (c *compiler) compileRegexpLiteral(v *ast.RegExpLiteral) compiledExpr {
	r := &compiledRegexpLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledCallExpr) emitGetter(putOnStack bool) {
	var calleeName unistring.String
	switch callee := e.callee.(type) {
	case *compiledDotExpr:
		callee.left.emitGetter(true)
		e.c.emit(dup)
		e.c.emit(getPropCallee(callee.name))
	case *compiledBracketExpr:
		callee.left.emitGetter(true)
		e.c.emit(dup)
		callee.member.emitGetter(true)
		e.c.emit(getElemCallee)
	case *compiledIdentifierExpr:
		calleeName = callee.name
		callee.emitGetterAndCallee()
	default:
		e.c.emit(loadUndef)
		callee.emitGetter(true)
	}

	for _, expr := range e.args {
		expr.emitGetter(true)
	}

	e.addSrcMap()
	if calleeName == "eval" {
		foundFunc := false
		for sc := e.c.scope; sc != nil; sc = sc.outer {
			if !foundFunc && sc.function {
				foundFunc = true
				sc.thisNeeded, sc.argsNeeded = true, true
				if !sc.strict {
					sc.dynamic = true
				}
			}
			sc.dynLookup = true
		}

		if e.c.scope.strict {
			e.c.emit(callEvalStrict(len(e.args)))
		} else {
			e.c.emit(callEval(len(e.args)))
		}
	} else {
		e.c.emit(call(len(e.args)))
	}

	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledCallExpr) deleteExpr() compiledExpr {
	r := &defaultDeleteExpr{
		expr: e,
	}
	r.init(e.c, file.Idx(e.offset+1))
	return r
}

func (c *compiler) compileCallExpression(v *ast.CallExpression) compiledExpr {

	args := make([]compiledExpr, len(v.ArgumentList))
	for i, argExpr := range v.ArgumentList {
		args[i] = c.compileExpression(argExpr)
	}

	r := &compiledCallExpr{
		args:   args,
		callee: c.compileExpression(v.Callee),
	}
	r.init(c, v.LeftParenthesis)
	return r
}

func (c *compiler) compileIdentifierExpression(v *ast.Identifier) compiledExpr {
	if c.scope.strict {
		c.checkIdentifierName(v.Name, int(v.Idx)-1)
	}

	r := &compiledIdentifierExpr{
		name: v.Name,
	}
	r.offset = int(v.Idx) - 1
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileNumberLiteral(v *ast.NumberLiteral) compiledExpr {
	if c.scope.strict && octalRegexp.MatchString(v.Literal) {
		c.throwSyntaxError(int(v.Idx)-1, "Octal literals are not allowed in strict mode")
		panic("Unreachable")
	}
	var val Value
	switch num := v.Value.(type) {
	case int64:
		val = intToValue(num)
	case float64:
		val = floatToValue(num)
	default:
		panic(fmt.Errorf("Unsupported number literal type: %T", v.Value))
	}
	r := &compiledLiteral{
		val: val,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileStringLiteral(v *ast.StringLiteral) compiledExpr {
	r := &compiledLiteral{
		val: stringValueFromRaw(v.Value),
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileBooleanLiteral(v *ast.BooleanLiteral) compiledExpr {
	var val Value
	if v.Value {
		val = valueTrue
	} else {
		val = valueFalse
	}

	r := &compiledLiteral{
		val: val,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileAssignExpression(v *ast.AssignExpression) compiledExpr {
	// log.Printf("compileAssignExpression(): %+v", v)

	r := &compiledAssignExpr{
		left:     c.compileExpression(v.Left),
		right:    c.compileExpression(v.Right),
		operator: v.Operator,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledEnumGetExpr) emitGetter(putOnStack bool) {
	e.c.emit(enumGet)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileObjectAssignmentPattern(v *ast.ObjectPattern) compiledExpr {
	r := &compiledObjectAssignmentPattern{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledObjectAssignmentPattern) emitGetter(putOnStack bool) {
	if putOnStack {
		e.c.emit(loadUndef)
	}
}

func (c *compiler) compileArrayAssignmentPattern(v *ast.ArrayPattern) compiledExpr {
	r := &compiledArrayAssignmentPattern{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledArrayAssignmentPattern) emitGetter(putOnStack bool) {
	if putOnStack {
		e.c.emit(loadUndef)
	}
}

func (c *compiler) emitNamed(expr compiledExpr, name unistring.String) {
	if en, ok := expr.(interface {
		emitNamed(name unistring.String)
	}); ok {
		en.emitNamed(name)
	} else {
		expr.emitGetter(true)
	}
}

func (e *compiledFunctionLiteral) emitNamed(name unistring.String) {
	e.lhsName = name
	e.emitGetter(true)
}

func (c *compiler) emitPattern(pattern ast.Pattern, emitter func(target, init compiledExpr), putOnStack bool) {
	switch pattern := pattern.(type) {
	case *ast.ObjectPattern:
		c.emitObjectPattern(pattern, emitter, putOnStack)
	case *ast.ArrayPattern:
		c.emitArrayPattern(pattern, emitter, putOnStack)
	default:
		panic(fmt.Errorf("unsupported Pattern: %T", pattern))
	}
}

func (c *compiler) emitAssign(target ast.Expression, init compiledExpr, emitAssignSimple func(target, init compiledExpr)) {
	pattern, isPattern := target.(ast.Pattern)
	if isPattern {
		init.emitGetter(true)
		c.emitPattern(pattern, emitAssignSimple, false)
	} else {
		emitAssignSimple(c.compileExpression(target), init)
	}
}

func (c *compiler) emitObjectPattern(pattern *ast.ObjectPattern, emitAssign func(target, init compiledExpr), putOnStack bool) {
	if pattern.Rest != nil {
		c.emit(createDestructSrc)
	} else {
		c.emit(checkObjectCoercible)
	}
	for _, prop := range pattern.Properties {
		switch prop := prop.(type) {
		case *ast.PropertyShort:
			c.emit(dup)
			emitAssign(c.compileIdentifierExpression(&prop.Name), c.compilePatternInitExpr(func() {
				c.emit(getProp(prop.Name.Name))
			}, prop.Initializer, prop.Idx0()))
		case *ast.PropertyKeyed:
			c.emit(dup)
			c.compileExpression(prop.Key).emitGetter(true)
			var target ast.Expression
			var initializer ast.Expression
			if e, ok := prop.Value.(*ast.AssignExpression); ok {
				target = e.Left
				initializer = e.Right
			} else {
				target = prop.Value
			}
			c.emitAssign(target, c.compilePatternInitExpr(func() {
				c.emit(getElem)
			}, initializer, prop.Idx0()), emitAssign)
		default:
			c.throwSyntaxError(int(prop.Idx0()-1), "Unsupported AssignmentProperty type: %T", prop)
		}
	}
	if pattern.Rest != nil {
		emitAssign(c.compileExpression(pattern.Rest), c.compileEmitterExpr(func() {
			c.emit(copyRest)
		}, pattern.Rest.Idx0()))
		c.emit(pop)
	}
	if !putOnStack {
		c.emit(pop)
	}
}

func (c *compiler) emitArrayPattern(pattern *ast.ArrayPattern, emitAssign func(target, init compiledExpr), putOnStack bool) {
	var marks []int
	c.emit(iterate)
	for _, elt := range pattern.Elements {
		switch elt := elt.(type) {
		case nil:
			marks = append(marks, len(c.p.code))
			c.emit(nil)
		case *ast.Identifier:
			emitAssign(c.compileIdentifierExpression(elt), c.compilePatternInitExpr(func() {
				marks = append(marks, len(c.p.code))
				c.emit(nil, enumGet)
			}, nil, elt.Idx0()))
		case *ast.AssignExpression:
			c.emitAssign(elt.Left, c.compilePatternInitExpr(func() {
				marks = append(marks, len(c.p.code))
				c.emit(nil, enumGet)
			}, elt.Right, elt.Idx0()), emitAssign)
		case ast.Pattern:
			c.compilePatternInitExpr(func() {
				marks = append(marks, len(c.p.code))
				c.emit(nil, enumGet)
			}, nil, elt.Idx0()).emitGetter(true)
			c.emitPattern(elt, emitAssign, false)
		case *ast.DotExpression:
			emitAssign(c.compileDotExpr(elt), c.compilePatternInitExpr(func() {
				marks = append(marks, len(c.p.code))
				c.emit(nil, enumGet)
			}, nil, elt.Idx0()))

		case *ast.BracketExpression:
			emitAssign(c.compileBracketExpr(elt), c.compilePatternInitExpr(func() {
				marks = append(marks, len(c.p.code))
				c.emit(nil, enumGet)
			}, nil, elt.Idx0()))
		default:
			c.throwSyntaxError(int(elt.Idx0()-1), "Unsupported AssignmentProperty type: %T", elt)
		}
	}
	if pattern.Rest != nil {
		c.emitAssign(pattern.Rest, c.compileEmitterExpr(func() {
			c.emit(newArrayFromIter)
		}, pattern.Rest.Idx0()), emitAssign)
	} else {
		c.emit(enumPopClose)
	}
	mark1 := len(c.p.code)
	c.emit(nil)

	for i, elt := range pattern.Elements {
		switch elt := elt.(type) {
		case nil:
			c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
		case *ast.Identifier:
			emitAssign(c.compileIdentifierExpression(elt), c.compileEmitterExpr(func() {
				c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
				c.emit(loadUndef)
			}, elt.Idx0()))
		case *ast.AssignExpression:
			c.emitAssign(elt.Left, c.compileNamedEmitterExpr(func(name unistring.String) {
				c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
				c.emitNamed(c.compileExpression(elt.Right), name)
			}, elt.Idx0()), emitAssign)
		case ast.Pattern:
			c.compilePatternInitExpr(func() {
				c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
				c.emit(loadUndef)
			}, nil, elt.Idx0()).emitGetter(true)
			c.emitPattern(elt, emitAssign, false)
		case *ast.DotExpression:
			emitAssign(c.compileDotExpr(elt), c.compileEmitterExpr(func() {
				c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
				c.emit(loadUndef)
			}, elt.Idx0()))
		case *ast.BracketExpression:
			emitAssign(c.compileBracketExpr(elt), c.compilePatternInitExpr(func() {
				c.p.code[marks[i]] = iterNext(len(c.p.code) - marks[i])
				c.emit(loadUndef)
			}, nil, elt.Idx0()))
		default:
			c.throwSyntaxError(int(elt.Idx0()-1), "Unsupported AssignmentProperty type: %T", elt)
		}
	}
	c.emit(enumPop)
	if pattern.Rest != nil {
		c.emitAssign(pattern.Rest, c.compileExpression(
			&ast.ArrayLiteral{
				LeftBracket:  pattern.Rest.Idx0(),
				RightBracket: pattern.Rest.Idx0(),
			}), emitAssign)
	}
	c.p.code[mark1] = jump(len(c.p.code) - mark1)

	if !putOnStack {
		c.emit(pop)
	}
}

func (e *compiledObjectAssignmentPattern) emitSetter(valueExpr compiledExpr, putOnStack bool) {
	valueExpr.emitGetter(true)
	e.c.emitObjectPattern(e.expr, e.c.emitPatternAssign, putOnStack)
}

func (e *compiledArrayAssignmentPattern) emitSetter(valueExpr compiledExpr, putOnStack bool) {
	valueExpr.emitGetter(true)
	e.c.emitArrayPattern(e.expr, e.c.emitPatternAssign, putOnStack)
}

type compiledPatternInitExpr struct {
	baseCompiledExpr
	emitSrc func()
	def     compiledExpr
}

func (e *compiledPatternInitExpr) emitGetter(putOnStack bool) {
	if !putOnStack {
		return
	}
	e.emitSrc()
	if e.def != nil {
		mark := len(e.c.p.code)
		e.c.emit(nil)
		e.def.emitGetter(true)
		e.c.p.code[mark] = jdef(len(e.c.p.code) - mark)
	}
}

func (e *compiledPatternInitExpr) emitNamed(name unistring.String) {
	e.emitSrc()
	if e.def != nil {
		mark := len(e.c.p.code)
		e.c.emit(nil)
		e.c.emitNamed(e.def, name)
		e.c.p.code[mark] = jdef(len(e.c.p.code) - mark)
	}
}

func (c *compiler) compilePatternInitExpr(emitSrc func(), def ast.Expression, idx file.Idx) compiledExpr {
	r := &compiledPatternInitExpr{
		emitSrc: emitSrc,
		def:     c.compileExpression(def),
	}
	r.init(c, idx)
	return r
}

type compiledEmitterExpr struct {
	baseCompiledExpr
	emitter      func()
	namedEmitter func(name unistring.String)
}

func (e *compiledEmitterExpr) emitGetter(putOnStack bool) {
	if e.emitter != nil {
		e.emitter()
	} else {
		e.namedEmitter("")
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledEmitterExpr) emitNamed(name unistring.String) {
	if e.namedEmitter != nil {
		e.namedEmitter(name)
	} else {
		e.emitter()
	}
}

func (c *compiler) compileEmitterExpr(emitter func(), idx file.Idx) *compiledEmitterExpr {
	r := &compiledEmitterExpr{
		emitter: emitter,
	}
	r.init(c, idx)
	return r
}

func (c *compiler) compileNamedEmitterExpr(namedEmitter func(unistring.String), idx file.Idx) *compiledEmitterExpr {
	r := &compiledEmitterExpr{
		namedEmitter: namedEmitter,
	}
	r.init(c, idx)
	return r
}

func (c *compiler) compileDotExpr(idx *ast.DotExpression) *compiledDotExpr {
	r := &compiledDotExpr{
		left: c.compileExpression(idx.Left),
		name: idx.Identifier.Name,
	}
	r.init(c, idx.Idx0())
	return r
}

func (c *compiler) compileBracketExpr(idx *ast.BracketExpression) *compiledBracketExpr {
	r := &compiledBracketExpr{
		left:   c.compileExpression(idx.Left),
		member: c.compileExpression(idx.Member),
	}
	r.init(c, idx.Idx0())
	return r
}
