package parser

import (
	"fmt"

	"github.com/nitrogen-lang/nitrogen/src/ast"
	"github.com/nitrogen-lang/nitrogen/src/token"
)

func (p *Parser) parseFunctionLiteral() ast.Expression {
	if p.settings.Debug {
		fmt.Println("parseFunctionLiteral")
	}
	lit := &ast.FunctionLiteral{
		Token:  p.curToken,
		Name:   "(anonymous)",
		FQName: "(anonymous)",
	}

	if !p.expectPeek(token.LParen) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBrace) {
		return nil
	}

	lit.Body = p.parseBlockStatements()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	if p.settings.Debug {
		fmt.Println("parseFunctionParameters")
	}
	idents := []*ast.Identifier{}

	if p.peekTokenIs(token.RParen) {
		p.nextToken()
		return idents
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	idents = append(idents, ident)

	for p.peekTokenIs(token.Comma) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		idents = append(idents, ident)
	}

	if !p.expectPeek(token.RParen) {
		return nil
	}

	return idents
}

func (p *Parser) parseCallExpression(left ast.Expression) ast.Node {
	if p.settings.Debug {
		fmt.Println("parseCallExpression")
	}
	return &ast.CallExpression{
		Token:     p.curToken,
		Function:  left,
		Arguments: p.parseExpressionList(token.RParen),
	}
}
