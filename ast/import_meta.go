package ast

import "github.com/dop251/goja/file"

// ImportMeta is the import.meta expression.
type ImportMeta struct {
	Idx file.Idx
}

func (i *ImportMeta) Idx0() file.Idx { return i.Idx }

func (i *ImportMeta) Idx1() file.Idx { return i.Idx }

func (*ImportMeta) _expressionNode() {}
