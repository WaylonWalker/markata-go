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
				{Type: TOKEN_IDENTIFIER, Value: "published"},
				{Type: TOKEN_COMPARE_OP, Value: "=="},
				{Type: TOKEN_BOOL, Value: "True", Literal: true},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "'python' in tags",
			expected: []Token{
				{Type: TOKEN_STRING, Value: "python", Literal: "python"},
				{Type: TOKEN_IN, Value: "in"},
				{Type: TOKEN_IDENTIFIER, Value: "tags"},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "date <= today",
			expected: []Token{
				{Type: TOKEN_IDENTIFIER, Value: "date"},
				{Type: TOKEN_COMPARE_OP, Value: "<="},
				{Type: TOKEN_TODAY, Value: "today"},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "not draft",
			expected: []Token{
				{Type: TOKEN_NOT, Value: "not"},
				{Type: TOKEN_IDENTIFIER, Value: "draft"},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "title.startswith('How')",
			expected: []Token{
				{Type: TOKEN_IDENTIFIER, Value: "title"},
				{Type: TOKEN_DOT, Value: "."},
				{Type: TOKEN_IDENTIFIER, Value: "startswith"},
				{Type: TOKEN_LPAREN, Value: "("},
				{Type: TOKEN_STRING, Value: "How", Literal: "How"},
				{Type: TOKEN_RPAREN, Value: ")"},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "word_count > 400",
			expected: []Token{
				{Type: TOKEN_IDENTIFIER, Value: "word_count"},
				{Type: TOKEN_COMPARE_OP, Value: ">"},
				{Type: TOKEN_NUMBER, Value: "400", Literal: int64(400)},
				{Type: TOKEN_EOF},
			},
		},
		{
			input: "status == 'draft' or status == 'review'",
			expected: []Token{
				{Type: TOKEN_IDENTIFIER, Value: "status"},
				{Type: TOKEN_COMPARE_OP, Value: "=="},
				{Type: TOKEN_STRING, Value: "draft", Literal: "draft"},
				{Type: TOKEN_LOGIC_OP, Value: "or"},
				{Type: TOKEN_IDENTIFIER, Value: "status"},
				{Type: TOKEN_COMPARE_OP, Value: "=="},
				{Type: TOKEN_STRING, Value: "review", Literal: "review"},
				{Type: TOKEN_EOF},
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
			if tok.Type != TOKEN_NUMBER {
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
			if tok.Type != TOKEN_STRING {
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
		{"True", TOKEN_BOOL, boolPtr(true)},
		{"False", TOKEN_BOOL, boolPtr(false)},
		{"None", TOKEN_NONE, nil},
		{"and", TOKEN_LOGIC_OP, nil},
		{"or", TOKEN_LOGIC_OP, nil},
		{"not", TOKEN_NOT, nil},
		{"in", TOKEN_IN, nil},
		{"today", TOKEN_TODAY, nil},
		{"now", TOKEN_NOW, nil},
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
