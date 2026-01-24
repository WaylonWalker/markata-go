package filter

import (
	"fmt"
)

// Expr is the interface for all AST nodes
type Expr interface {
	exprNode()
	String() string
}

// BinaryExpr represents a binary expression (left op right)
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (e *BinaryExpr) exprNode() {}
func (e *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Op, e.Right.String())
}

// UnaryExpr represents a unary expression (op expr)
type UnaryExpr struct {
	Op   string
	Expr Expr
}

func (e *UnaryExpr) exprNode() {}
func (e *UnaryExpr) String() string {
	return fmt.Sprintf("(%s %s)", e.Op, e.Expr.String())
}

// InExpr represents an 'in' expression (value in collection)
type InExpr struct {
	Value      Expr
	Collection Expr
}

func (e *InExpr) exprNode() {}
func (e *InExpr) String() string {
	return fmt.Sprintf("(%s in %s)", e.Value.String(), e.Collection.String())
}

// CallExpr represents a method call (object.method(args...))
type CallExpr struct {
	Object Expr
	Method string
	Args   []Expr
}

func (e *CallExpr) exprNode() {}
func (e *CallExpr) String() string {
	args := ""
	for i, arg := range e.Args {
		if i > 0 {
			args += ", "
		}
		args += arg.String()
	}
	return fmt.Sprintf("%s.%s(%s)", e.Object.String(), e.Method, args)
}

// FieldAccess represents field access (object.field)
type FieldAccess struct {
	Object Expr
	Field  string
}

func (e *FieldAccess) exprNode() {}
func (e *FieldAccess) String() string {
	return fmt.Sprintf("%s.%s", e.Object.String(), e.Field)
}

// Identifier represents an identifier
type Identifier struct {
	Name string
}

func (e *Identifier) exprNode() {}
func (e *Identifier) String() string {
	return e.Name
}

// Literal represents a literal value (string, number, bool, nil)
type Literal struct {
	Value interface{}
}

func (e *Literal) exprNode() {}
func (e *Literal) String() string {
	if e.Value == nil {
		return "None"
	}
	return fmt.Sprintf("%v", e.Value)
}

// SpecialValue represents special values like 'today' and 'now'
type SpecialValue struct {
	Name string
}

func (e *SpecialValue) exprNode() {}
func (e *SpecialValue) String() string {
	return e.Name
}

// Parser parses filter expressions into an AST
type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
}

// NewParser creates a new parser for the given input
func NewParser(input string) (*Parser, error) {
	p := &Parser{
		lexer: NewLexer(input),
	}
	// Read two tokens to initialize current and peek
	var err error
	p.current, err = p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	p.peek, err = p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// advance moves to the next token
func (p *Parser) advance() error {
	p.current = p.peek
	var err error
	p.peek, err = p.lexer.NextToken()
	return err
}

// Parse parses the input and returns the AST
func (p *Parser) Parse() (Expr, error) {
	if p.current.Type == TokenEOF {
		// Empty expression - always matches
		return &Literal{Value: true}, nil
	}
	return p.parseOr()
}

// parseOr parses 'or' expressions (lowest precedence)
func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenLogicOp && p.current.Value == "or" {
		op := p.current.Value
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}

	return left, nil
}

// parseAnd parses 'and' expressions
func (p *Parser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenLogicOp && p.current.Value == "and" {
		op := p.current.Value
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}

	return left, nil
}

// parseNot parses 'not' expressions
func (p *Parser) parseNot() (Expr, error) {
	if p.current.Type == TokenNot {
		if err := p.advance(); err != nil {
			return nil, err
		}
		expr, err := p.parseNot() // Allow chained not
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "not", Expr: expr}, nil
	}
	return p.parseComparison()
}

// parseComparison parses comparison and 'in'/'contains' expressions
func (p *Parser) parseComparison() (Expr, error) {
	left, err := p.parseAccess()
	if err != nil {
		return nil, err
	}

	// Handle 'in' expressions
	if p.current.Type == TokenIn {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAccess()
		if err != nil {
			return nil, err
		}
		return &InExpr{Value: left, Collection: right}, nil
	}

	// Handle 'contains' expressions (legacy syntax: "field contains value" → "value in field")
	if p.current.Type == TokenContains {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAccess()
		if err != nil {
			return nil, err
		}
		// Swap: "field contains value" becomes "value in field"
		return &InExpr{Value: right, Collection: left}, nil
	}

	// Handle comparison operators
	if p.current.Type == TokenCompareOp {
		op := p.current.Value
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAccess()
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Op: op, Right: right}, nil
	}

	return left, nil
}

// parseAccess parses dot access and method calls
func (p *Parser) parseAccess() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.current.Type == TokenDot {
		if err := p.advance(); err != nil {
			return nil, err
		}

		// Accept identifier or 'contains' keyword as method name (for backward compatibility)
		if p.current.Type != TokenIdentifier && p.current.Type != TokenContains {
			return nil, fmt.Errorf("expected identifier after '.', got %s", p.current)
		}

		name := p.current.Value
		if err := p.advance(); err != nil {
			return nil, err
		}

		// Check if it's a method call
		if p.current.Type == TokenLParen {
			if err := p.advance(); err != nil {
				return nil, err
			}

			var args []Expr
			if p.current.Type != TokenRParen {
				for {
					arg, err := p.parseOr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)

					if p.current.Type != TokenComma {
						break
					}
					if err := p.advance(); err != nil {
						return nil, err
					}
				}
			}

			if p.current.Type != TokenRParen {
				return nil, fmt.Errorf("expected ')' after method arguments, got %s", p.current)
			}
			if err := p.advance(); err != nil {
				return nil, err
			}

			expr = &CallExpr{Object: expr, Method: name, Args: args}
		} else {
			expr = &FieldAccess{Object: expr, Field: name}
		}
	}

	return expr, nil
}

// parsePrimary parses primary expressions (literals, identifiers, parenthesized expressions)
func (p *Parser) parsePrimary() (Expr, error) {
	switch p.current.Type {
	case TokenIdentifier:
		name := p.current.Value
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &Identifier{Name: name}, nil

	case TokenString:
		value := p.current.Literal
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &Literal{Value: value}, nil

	case TokenNumber:
		value := p.current.Literal
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &Literal{Value: value}, nil

	case TokenBool:
		value := p.current.Literal
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &Literal{Value: value}, nil

	case TokenNone:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &Literal{Value: nil}, nil

	case TokenToday:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &SpecialValue{Name: "today"}, nil

	case TokenNow:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &SpecialValue{Name: "now"}, nil

	case TokenLParen:
		if err := p.advance(); err != nil {
			return nil, err
		}
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.current.Type != TokenRParen {
			return nil, fmt.Errorf("expected ')' after expression, got %s", p.current)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token: %s", p.current)
	}
}

// ParseExpression parses a filter expression string into an AST
func ParseExpression(input string) (Expr, error) {
	parser, err := NewParser(input)
	if err != nil {
		return nil, err
	}
	return parser.Parse()
}
