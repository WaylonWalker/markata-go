package filter

import (
	"testing"
)

func TestParser_SimpleLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"True", "true"},
		{"False", "false"},
		{"None", "None"},
		{"42", "42"},
		{"'hello'", "hello"},
		{"today", "today"},
		{"now", "now"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_Comparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"published == True", "(published == true)"},
		{"draft != False", "(draft != false)"},
		{"count > 10", "(count > 10)"},
		{"count < 100", "(count < 100)"},
		{"count >= 5", "(count >= 5)"},
		{"count <= 50", "(count <= 50)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_InExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"'python' in tags", "(python in tags)"},
		{"tag in categories", "(tag in categories)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_NotExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"not draft", "(not draft)"},
		{"not skip", "(not skip)"},
		{"not not published", "(not (not published))"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_LogicalExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a and b", "(a and b)"},
		{"a or b", "(a or b)"},
		{"a and b and c", "((a and b) and c)"},
		{"a or b or c", "((a or b) or c)"},
		{"a and b or c", "((a and b) or c)"},
		{"a or b and c", "(a or (b and c))"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_MethodCall(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"title.startswith('How')", "title.startswith(How)"},
		{"name.endswith('.md')", "name.endswith(.md)"},
		{"text.lower()", "text.lower()"},
		{"s.contains('test')", "s.contains(test)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_FieldAccess(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"post.title", "post.title"},
		{"meta.author.name", "meta.author.name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_Parentheses(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(a)", "a"},
		{"(a and b)", "(a and b)"},
		{"(a or b) and c", "((a or b) and c)"},
		{"a and (b or c)", "(a and (b or c))"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_ComplexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"published == True and date <= today",
			"((published == true) and (date <= today))",
		},
		{
			"status == 'draft' or status == 'review'",
			"((status == draft) or (status == review))",
		},
		{
			"published == True and 'python' in tags",
			"((published == true) and (python in tags))",
		},
		{
			"not draft and published == True",
			"((not draft) and (published == true))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expr.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.String())
			}
		})
	}
}

func TestParser_EmptyExpression(t *testing.T) {
	expr, err := ParseExpression("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty expression should evaluate to true (always match)
	lit, ok := expr.(*Literal)
	if !ok {
		t.Fatalf("expected *Literal, got %T", expr)
	}
	if lit.Value != true {
		t.Errorf("expected true, got %v", lit.Value)
	}
}
