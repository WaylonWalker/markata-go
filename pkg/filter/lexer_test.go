package filter

import (
	"testing"
)

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: "published == True",
			expected: []Token{
				{Type: TokenIdentifier, Value: "published"},
				{Type: TokenCompareOp, Value: "=="},
				{Type: TokenBool, Value: "True", Literal: true},
				{Type: TokenEOF},
			},
		},
		{
			input: "'python' in tags",
			expected: []Token{
				{Type: TokenString, Value: "python", Literal: "python"},
				{Type: TokenIn, Value: "in"},
				{Type: TokenIdentifier, Value: "tags"},
				{Type: TokenEOF},
			},
		},
		{
			input: "date <= today",
			expected: []Token{
				{Type: TokenIdentifier, Value: "date"},
				{Type: TokenCompareOp, Value: "<="},
				{Type: TokenToday, Value: "today"},
				{Type: TokenEOF},
			},
		},
		{
			input: "not draft",
			expected: []Token{
				{Type: TokenNot, Value: "not"},
				{Type: TokenIdentifier, Value: "draft"},
				{Type: TokenEOF},
			},
		},
		{
			input: "title.startswith('How')",
			expected: []Token{
				{Type: TokenIdentifier, Value: "title"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdentifier, Value: "startswith"},
				{Type: TokenLParen, Value: "("},
				{Type: TokenString, Value: "How", Literal: "How"},
				{Type: TokenRParen, Value: ")"},
				{Type: TokenEOF},
			},
		},
		{
			input: "word_count > 400",
			expected: []Token{
				{Type: TokenIdentifier, Value: "word_count"},
				{Type: TokenCompareOp, Value: ">"},
				{Type: TokenNumber, Value: "400", Literal: int64(400)},
				{Type: TokenEOF},
			},
		},
		{
			input: "status == 'draft' or status == 'review'",
			expected: []Token{
				{Type: TokenIdentifier, Value: "status"},
				{Type: TokenCompareOp, Value: "=="},
				{Type: TokenString, Value: "draft", Literal: "draft"},
				{Type: TokenLogicOp, Value: "or"},
				{Type: TokenIdentifier, Value: "status"},
				{Type: TokenCompareOp, Value: "=="},
				{Type: TokenString, Value: "review", Literal: "review"},
				{Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, tok := range tokens {
				exp := tt.expected[i]
				if tok.Type != exp.Type {
					t.Errorf("token[%d]: expected type %v, got %v", i, exp.Type, tok.Type)
				}
				if tok.Value != exp.Value {
					t.Errorf("token[%d]: expected value %q, got %q", i, exp.Value, tok.Value)
				}
				if exp.Literal != nil && tok.Literal != exp.Literal {
					t.Errorf("token[%d]: expected literal %v, got %v", i, exp.Literal, tok.Literal)
				}
			}
		})
	}
}

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		input   string
		literal interface{}
	}{
		{"42", int64(42)},
		{"3.14", float64(3.14)},
		{"-10", int64(-10)},
		{"-2.5", float64(-2.5)},
		{"0", int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.Type != TokenNumber {
				t.Errorf("expected NUMBER, got %v", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %v (%T), got %v (%T)", tt.literal, tt.literal, tok.Literal, tok.Literal)
			}
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`"with spaces"`, "with spaces"},
		{`"escaped\"quote"`, `escaped"quote`},
		{`'escaped\'quote'`, `escaped'quote`},
		{`"newline\nhere"`, "newline\nhere"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.Type != TokenString {
				t.Errorf("expected STRING, got %v", tok.Type)
			}
			if tok.Literal != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tok.Literal)
			}
		})
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		input       string
		tokenType   TokenType
		boolLiteral *bool
	}{
		{"True", TokenBool, boolPtr(true)},
		{"False", TokenBool, boolPtr(false)},
		{"None", TokenNone, nil},
		{"and", TokenLogicOp, nil},
		{"or", TokenLogicOp, nil},
		{"not", TokenNot, nil},
		{"in", TokenIn, nil},
		{"today", TokenToday, nil},
		{"now", TokenNow, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok.Type != tt.tokenType {
				t.Errorf("expected %v, got %v", tt.tokenType, tok.Type)
			}
			if tt.boolLiteral != nil {
				if tok.Literal != *tt.boolLiteral {
					t.Errorf("expected literal %v, got %v", *tt.boolLiteral, tok.Literal)
				}
			}
		})
	}
}

func TestLexer_Errors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`"unterminated`},
		{`'unterminated`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			_, err := lexer.Tokenize()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
