package filter

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_IDENTIFIER
	TOKEN_STRING
	TOKEN_NUMBER
	TOKEN_BOOL
	TOKEN_NONE
	TOKEN_COMPARE_OP // ==, !=, <, >, <=, >=
	TOKEN_LOGIC_OP   // and, or
	TOKEN_IN
	TOKEN_NOT
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_DOT
	TOKEN_COMMA
	TOKEN_TODAY
	TOKEN_NOW
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
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_STRING:
		return fmt.Sprintf("STRING(%q)", t.Value)
	case TOKEN_NUMBER:
		return fmt.Sprintf("NUMBER(%v)", t.Literal)
	case TOKEN_BOOL:
		return fmt.Sprintf("BOOL(%v)", t.Literal)
	case TOKEN_NONE:
		return "NONE"
	default:
		return fmt.Sprintf("%s(%s)", tokenTypeName(t.Type), t.Value)
	}
}

func tokenTypeName(t TokenType) string {
	names := map[TokenType]string{
		TOKEN_EOF:        "EOF",
		TOKEN_IDENTIFIER: "IDENTIFIER",
		TOKEN_STRING:     "STRING",
		TOKEN_NUMBER:     "NUMBER",
		TOKEN_BOOL:       "BOOL",
		TOKEN_NONE:       "NONE",
		TOKEN_COMPARE_OP: "COMPARE_OP",
		TOKEN_LOGIC_OP:   "LOGIC_OP",
		TOKEN_IN:         "IN",
		TOKEN_NOT:        "NOT",
		TOKEN_LPAREN:     "LPAREN",
		TOKEN_RPAREN:     "RPAREN",
		TOKEN_DOT:        "DOT",
		TOKEN_COMMA:      "COMMA",
		TOKEN_TODAY:      "TODAY",
		TOKEN_NOW:        "NOW",
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
		return Token{Type: TOKEN_EOF, Pos: pos}, nil

	case '(':
		l.readChar()
		return Token{Type: TOKEN_LPAREN, Value: "(", Pos: pos}, nil

	case ')':
		l.readChar()
		return Token{Type: TOKEN_RPAREN, Value: ")", Pos: pos}, nil

	case '.':
		l.readChar()
		return Token{Type: TOKEN_DOT, Value: ".", Pos: pos}, nil

	case ',':
		l.readChar()
		return Token{Type: TOKEN_COMMA, Value: ",", Pos: pos}, nil

	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TOKEN_COMPARE_OP, Value: "==", Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '=' at position %d, expected '=='", pos)

	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TOKEN_COMPARE_OP, Value: "!=", Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '!' at position %d, expected '!='", pos)

	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TOKEN_COMPARE_OP, Value: "<=", Pos: pos}, nil
		}
		l.readChar()
		return Token{Type: TOKEN_COMPARE_OP, Value: "<", Pos: pos}, nil

	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: TOKEN_COMPARE_OP, Value: ">=", Pos: pos}, nil
		}
		l.readChar()
		return Token{Type: TOKEN_COMPARE_OP, Value: ">", Pos: pos}, nil

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
	return Token{Type: TOKEN_STRING, Value: value, Literal: value, Pos: pos}, nil
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
		fmt.Sscanf(value, "%f", &f)
		literal = f
	} else {
		var i int64
		fmt.Sscanf(value, "%d", &i)
		literal = i
	}

	return Token{Type: TOKEN_NUMBER, Value: value, Literal: literal, Pos: pos}, nil
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
	case "True":
		return Token{Type: TOKEN_BOOL, Value: value, Literal: true, Pos: pos}, nil
	case "False":
		return Token{Type: TOKEN_BOOL, Value: value, Literal: false, Pos: pos}, nil
	case "None":
		return Token{Type: TOKEN_NONE, Value: value, Literal: nil, Pos: pos}, nil
	case "and":
		return Token{Type: TOKEN_LOGIC_OP, Value: value, Pos: pos}, nil
	case "or":
		return Token{Type: TOKEN_LOGIC_OP, Value: value, Pos: pos}, nil
	case "not":
		return Token{Type: TOKEN_NOT, Value: value, Pos: pos}, nil
	case "in":
		return Token{Type: TOKEN_IN, Value: value, Pos: pos}, nil
	case "today":
		return Token{Type: TOKEN_TODAY, Value: value, Pos: pos}, nil
	case "now":
		return Token{Type: TOKEN_NOW, Value: value, Pos: pos}, nil
	default:
		return Token{Type: TOKEN_IDENTIFIER, Value: value, Pos: pos}, nil
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
		if tok.Type == TOKEN_EOF {
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
