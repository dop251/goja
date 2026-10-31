package ast

import "github.com/dop251/goja/file"

// ImportExpr is an import() call expression.
type ImportExpr struct {
	Idx      file.Idx
	Argument Expression
}

func (i *ImportExpr) Idx0() file.Idx { return i.Idx }

func (i *ImportExpr) Idx1() file.Idx { return i.Idx }

func (*ImportExpr) _expressionNode() {}
