package parser

import (
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/token"
)

// ParseModule parses src using the module goal.
func ParseModule(filename, src string, options ...Option) (*ast.Program, error) {
	return ParseFile(nil, filename, src, Module, options...)
}

func (self *_parser) parseImportDeclaration() ast.Statement {
	idx := self.idx
	self.next()
	decl := &ast.ImportDeclaration{Idx: idx}
	if self.token == token.STRING {
		if lit, ok := self.parsePrimaryExpression().(*ast.StringLiteral); ok {
			decl.Source = lit
		}
	} else {
		if self.token == token.IDENTIFIER {
			id := self.parseIdentifier()
			decl.Specifiers = append(decl.Specifiers, &ast.ImportSpecifier{Idx: id.Idx, Kind: ast.ImportDefault, Local: id})
			if self.token == token.COMMA {
				self.next()
			}
		}
		if self.token == token.MULTIPLY {
			self.next()
			self.next() // as
			id := self.parseIdentifier()
			decl.Specifiers = append(decl.Specifiers, &ast.ImportSpecifier{Idx: id.Idx, Kind: ast.ImportNamespace, Local: id})
		} else if self.token == token.LEFT_BRACE {
			self.next()
			for self.token != token.RIGHT_BRACE && self.token != token.EOF {
				imported := self.parseIdentifier()
				local := imported
				if self.literal == "as" {
					self.next()
					local = self.parseIdentifier()
				}
				decl.Specifiers = append(decl.Specifiers, &ast.ImportSpecifier{Idx: imported.Idx, Kind: ast.ImportNamed, Imported: imported, Local: local})
				if self.token == token.COMMA {
					self.next()
					continue
				}
				break
			}
			self.expect(token.RIGHT_BRACE)
		}
		self.next() // from
		if lit, ok := self.parsePrimaryExpression().(*ast.StringLiteral); ok {
			decl.Source = lit
		}
	}
	return decl
}

func (self *_parser) parseExportDeclaration() ast.Statement {
	idx := self.idx
	self.next()
	decl := &ast.ExportDeclaration{Idx: idx}
	if self.token == token.DEFAULT {
		self.next()
		decl.IsDefault = true
		decl.Declaration = self.parseExpression()
		return decl
	}
	if self.token == token.MULTIPLY {
		self.next()
		if self.literal == "as" {
			self.next()
			decl.Namespace = self.parseIdentifier()
		}
		self.next() // from
		if lit, ok := self.parsePrimaryExpression().(*ast.StringLiteral); ok {
			decl.Source = lit
		}
		return decl
	}
	if self.token == token.LEFT_BRACE {
		self.next()
		for self.token != token.RIGHT_BRACE && self.token != token.EOF {
			local := self.parseIdentifier()
			exported := local
			if self.literal == "as" {
				self.next()
				exported = self.parseIdentifier()
			}
			decl.Specifiers = append(decl.Specifiers, &ast.ExportSpecifier{Idx: local.Idx, Local: local, Exported: exported})
			if self.token == token.COMMA {
				self.next()
				continue
			}
			break
		}
		self.expect(token.RIGHT_BRACE)
		if self.literal == "from" {
			self.next()
			if lit, ok := self.parsePrimaryExpression().(*ast.StringLiteral); ok {
				decl.Source = lit
			}
		}
		return decl
	}
	decl.Declaration = self.parseStatement()
	return decl
}
