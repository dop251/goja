package ast

import "github.com/dop251/goja/file"

// ImportSpecifierKind describes the form of an import specifier.
type ImportSpecifierKind int

const (
	ImportDefault ImportSpecifierKind = iota
	ImportNamespace
	ImportNamed
)

// ImportSpecifier is one binding in an import declaration.
type ImportSpecifier struct {
	Idx      file.Idx
	Kind     ImportSpecifierKind
	Imported *Identifier
	Local    *Identifier
}

func (s *ImportSpecifier) Idx0() file.Idx { return s.Idx }

func (s *ImportSpecifier) Idx1() file.Idx {
	if s.Local != nil {
		return s.Local.Idx1()
	}
	return s.Idx
}

// ImportDeclaration is an import statement.
type ImportDeclaration struct {
	Idx        file.Idx
	Specifiers []*ImportSpecifier
	Source     *StringLiteral
}

func (d *ImportDeclaration) Idx0() file.Idx { return d.Idx }

func (d *ImportDeclaration) Idx1() file.Idx {
	if d.Source != nil {
		return d.Source.Idx1()
	}
	return d.Idx
}

func (*ImportDeclaration) _statementNode() {}

// ExportSpecifier is one binding in an export list.
type ExportSpecifier struct {
	Idx      file.Idx
	Local    *Identifier
	Exported *Identifier
}

func (s *ExportSpecifier) Idx0() file.Idx { return s.Idx }

func (s *ExportSpecifier) Idx1() file.Idx {
	if s.Exported != nil {
		return s.Exported.Idx1()
	}
	return s.Idx
}

// ExportDeclaration is an export statement.
type ExportDeclaration struct {
	Idx         file.Idx
	Specifiers  []*ExportSpecifier
	Source      *StringLiteral
	Declaration Node
	Namespace   *Identifier
	IsDefault   bool
}

func (d *ExportDeclaration) Idx0() file.Idx { return d.Idx }

func (d *ExportDeclaration) Idx1() file.Idx {
	if d.Source != nil {
		return d.Source.Idx1()
	}
	if d.Declaration != nil {
		return d.Declaration.Idx1()
	}
	return d.Idx
}

func (*ExportDeclaration) _statementNode() {}
