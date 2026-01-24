package filter

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenBool
	TokenNone
	TokenCompareOp // ==, !=, <, >, <=, >=
	TokenLogicOp   // and, or
	TokenIn
	TokenContains // contains (legacy syntax support)
	TokenNot
	TokenLParen
	TokenRParen
	TokenDot
	TokenComma
	TokenToday
	TokenNow
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Value   string
	Literal interface{} // For STRING, NUMBER, BOOL: the actual value
	Pos     int         // Position in the input string
}

// String returns a string representation of the token
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenString:
		return fmt.Sprintf("STRING(%q)", t.Value)
	case TokenNumber:
		return fmt.Sprintf("NUMBER(%v)", t.Literal)
	case TokenBool:
		return fmt.Sprintf("BOOL(%v)", t.Literal)
	case TokenNone:
		return "NONE"
	default:
		return fmt.Sprintf("%s(%s)", tokenTypeName(t.Type), t.Value)
	}
}

func tokenTypeName(t TokenType) string {
	names := map[TokenType]string{
		TokenEOF:        "EOF",
		TokenIdentifier: "IDENTIFIER",
		TokenString:     "STRING",
		TokenNumber:     "NUMBER",
		TokenBool:       "BOOL",
		TokenNone:       "NONE",
		TokenCompareOp:  "COMPARE_OP",
		TokenLogicOp:    "LOGIC_OP",
		TokenIn:         "IN",
		TokenContains:   "CONTAINS",
		TokenNot:        "NOT",
		TokenLParen:     "LPAREN",
		TokenRParen:     "RPAREN",
		TokenDot:        "DOT",
		TokenComma:      "COMMA",
		TokenToday:      "TODAY",
		TokenNow:        "NOW",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Lexer tokenizes a filter expression
type Lexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar reads the next character
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// skipWhitespace skips over whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	pos := l.pos

	switch l.ch {
	case 0:
		return Token{Type: TokenEOF, Pos: pos}, nil

	case '(':
		l.readChar()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}, nil

	case ')':
		l.readChar()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}, nil

	case '.':
		l.readChar()
		return Token{Type: TokenDot, Value: ".", Pos: pos}, nil

	case ',':
		l.readChar()
		return Token{Type: TokenComma, Value: ",", Pos: pos}, nil

	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenCompareOp, Value: "==", Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '=' at position %d, expected '=='", pos)

	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenCompareOp, Value: "!=", Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '!' at position %d, expected '!='", pos)

	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenCompareOp, Value: "<=", Pos: pos}, nil
		}
		l.readChar()
		return Token{Type: TokenCompareOp, Value: "<", Pos: pos}, nil

	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TokenCompareOp, Value: ">=", Pos: pos}, nil
		}
		l.readChar()
		return Token{Type: TokenCompareOp, Value: ">", Pos: pos}, nil

	case '"', '\'':
		return l.readString()

	default:
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			return l.readNumber()
		}
		if isLetter(l.ch) || l.ch == '_' {
			return l.readIdentifier()
		}
		return Token{}, fmt.Errorf("unexpected character '%c' at position %d", l.ch, pos)
	}
}

// readString reads a quoted string
func (l *Lexer) readString() (Token, error) {
	pos := l.pos
	quote := l.ch
	l.readChar() // consume opening quote

	var sb strings.Builder
	for l.ch != quote {
		if l.ch == 0 {
			return Token{}, fmt.Errorf("unterminated string starting at position %d", pos)
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}
	l.readChar() // consume closing quote

	value := sb.String()
	return Token{Type: TokenString, Value: value, Literal: value, Pos: pos}, nil
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() (Token, error) {
	pos := l.pos
	var sb strings.Builder

	// Handle negative sign
	if l.ch == '-' {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	// Read integer part
	for isDigit(l.ch) {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	// Check for decimal point
	isFloat := false
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		sb.WriteByte(l.ch)
		l.readChar()
		for isDigit(l.ch) {
			sb.WriteByte(l.ch)
			l.readChar()
		}
	}

	value := sb.String()
	var literal interface{}

	if isFloat {
		var f float64
		//nolint:errcheck // error checking not needed; lexer already validated the number format
		fmt.Sscanf(value, "%f", &f)
		literal = f
	} else {
		var i int64
		//nolint:errcheck // error checking not needed; lexer already validated the number format
		fmt.Sscanf(value, "%d", &i)
		literal = i
	}

	return Token{Type: TokenNumber, Value: value, Literal: literal, Pos: pos}, nil
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() (Token, error) {
	pos := l.pos
	var sb strings.Builder

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		sb.WriteByte(l.ch)
		l.readChar()
	}

	value := sb.String()

	// Check for keywords
	switch value {
	case "True", "true":
		return Token{Type: TokenBool, Value: value, Literal: true, Pos: pos}, nil
	case "False", "false":
		return Token{Type: TokenBool, Value: value, Literal: false, Pos: pos}, nil
	case "None":
		return Token{Type: TokenNone, Value: value, Literal: nil, Pos: pos}, nil
	case "and":
		return Token{Type: TokenLogicOp, Value: value, Pos: pos}, nil
	case "or":
		return Token{Type: TokenLogicOp, Value: value, Pos: pos}, nil
	case "not":
		return Token{Type: TokenNot, Value: value, Pos: pos}, nil
	case "in":
		return Token{Type: TokenIn, Value: value, Pos: pos}, nil
	case "contains":
		return Token{Type: TokenContains, Value: value, Pos: pos}, nil
	case "today":
		return Token{Type: TokenToday, Value: value, Pos: pos}, nil
	case "now":
		return Token{Type: TokenNow, Value: value, Pos: pos}, nil
	default:
		return Token{Type: TokenIdentifier, Value: value, Pos: pos}, nil
	}
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch))
}
