package filter

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Logical operator constants
const (
	opAnd = "and"
	opOr  = "or"
)

// EvalContext contains the context for evaluating filter expressions
type EvalContext struct {
	Today time.Time
	Now   time.Time
}

// NewEvalContext creates a new evaluation context with current time values
func NewEvalContext() *EvalContext {
	now := time.Now()
	// today is end of the current day (23:59:59.999999999) so that
	// "date <= today" includes all posts from today regardless of their
	// time component.  Previously this was midnight (00:00:00), which
	// caused posts dated today with any time component to be excluded.
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	return &EvalContext{
		Today: today,
		Now:   now,
	}
}

// Evaluate evaluates an expression against a post
func Evaluate(expr Expr, post *models.Post, ctx *EvalContext) (bool, error) {
	result, err := eval(expr, post, ctx)
	if err != nil {
		return false, err
	}
	return toBool(result), nil
}

// eval evaluates an expression and returns the result
func eval(expr Expr, post *models.Post, ctx *EvalContext) (interface{}, error) {
	switch e := expr.(type) {
	case *Literal:
		return e.Value, nil

	case *SpecialValue:
		switch e.Name {
		case "today":
			return ctx.Today, nil
		case "now":
			return ctx.Now, nil
		default:
			return nil, fmt.Errorf("unknown special value: %s", e.Name)
		}

	case *Identifier:
		return getField(post, e.Name)

	case *UnaryExpr:
		if e.Op == "not" {
			val, err := eval(e.Expr, post, ctx)
			if err != nil {
				return nil, err
			}
			return !toBool(val), nil
		}
		return nil, fmt.Errorf("unknown unary operator: %s", e.Op)

	case *BinaryExpr:
		return evalBinaryExpr(e, post, ctx)

	case *InExpr:
		return evalInExpr(e, post, ctx)

	case *CallExpr:
		return evalCallExpr(e, post, ctx)

	case *FieldAccess:
		obj, err := eval(e.Object, post, ctx)
		if err != nil {
			return nil, err
		}
		return getFieldFromValue(obj, e.Field)

	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// evalBinaryExpr evaluates a binary expression
func evalBinaryExpr(e *BinaryExpr, post *models.Post, ctx *EvalContext) (interface{}, error) {
	// Handle logical operators first (short-circuit evaluation)
	switch e.Op {
	case opAnd:
		left, err := eval(e.Left, post, ctx)
		if err != nil {
			return nil, err
		}
		if !toBool(left) {
			return false, nil
		}
		right, err := eval(e.Right, post, ctx)
		if err != nil {
			return nil, err
		}
		return toBool(right), nil

	case opOr:
		left, err := eval(e.Left, post, ctx)
		if err != nil {
			return nil, err
		}
		if toBool(left) {
			return true, nil
		}
		right, err := eval(e.Right, post, ctx)
		if err != nil {
			return nil, err
		}
		return toBool(right), nil
	}

	// Evaluate both sides for comparison operators
	left, err := eval(e.Left, post, ctx)
	if err != nil {
		return nil, err
	}
	right, err := eval(e.Right, post, ctx)
	if err != nil {
		return nil, err
	}

	// Handle comparison operators
	switch e.Op {
	case "==":
		return compare(left, right) == 0, nil
	case "!=":
		return compare(left, right) != 0, nil
	case "<":
		cmp := compare(left, right)
		return cmp < 0, nil
	case "<=":
		cmp := compare(left, right)
		return cmp <= 0, nil
	case ">":
		cmp := compare(left, right)
		return cmp > 0, nil
	case ">=":
		cmp := compare(left, right)
		return cmp >= 0, nil
	default:
		return nil, fmt.Errorf("unknown operator: %s", e.Op)
	}
}

// evalInExpr evaluates an 'in' expression
func evalInExpr(e *InExpr, post *models.Post, ctx *EvalContext) (interface{}, error) {
	value, err := eval(e.Value, post, ctx)
	if err != nil {
		return nil, err
	}

	// Backward compatibility: if value is nil and came from an identifier,
	// treat the identifier name as a string literal (for "tags contains go" syntax)
	if value == nil {
		if ident, ok := e.Value.(*Identifier); ok {
			value = ident.Name
		}
	}

	collection, err := eval(e.Collection, post, ctx)
	if err != nil {
		return nil, err
	}

	// Handle slice types
	rv := reflect.ValueOf(collection)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			item := rv.Index(i).Interface()
			if compare(value, item) == 0 {
				return true, nil
			}
		}
		return false, nil

	case reflect.String:
		// Check if value is in string
		valStr, ok := value.(string)
		if !ok {
			return false, nil
		}
		return strings.Contains(rv.String(), valStr), nil

	case reflect.Map:
		// Check if value is a key in the map
		valRv := reflect.ValueOf(value)
		if valRv.Type().AssignableTo(rv.Type().Key()) {
			return rv.MapIndex(valRv).IsValid(), nil
		}
		return false, nil

	default:
		return nil, fmt.Errorf("'in' operator requires a collection, got %T", collection)
	}
}

// evalCallExpr evaluates a method call expression
func evalCallExpr(e *CallExpr, post *models.Post, ctx *EvalContext) (interface{}, error) {
	obj, err := eval(e.Object, post, ctx)
	if err != nil {
		return nil, err
	}

	// Evaluate arguments
	args := make([]interface{}, len(e.Args))
	for i, arg := range e.Args {
		val, err := eval(arg, post, ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Handle string methods
	if str, ok := obj.(string); ok {
		return evalStringMethod(str, e.Method, args)
	}

	return nil, fmt.Errorf("cannot call method '%s' on type %T", e.Method, obj)
}

// evalStringMethod evaluates string methods
func evalStringMethod(s, method string, args []interface{}) (interface{}, error) {
	switch method {
	case "startswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("startswith() takes exactly 1 argument (%d given)", len(args))
		}
		prefix, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("startswith() argument must be a string")
		}
		return strings.HasPrefix(s, prefix), nil

	case "endswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("endswith() takes exactly 1 argument (%d given)", len(args))
		}
		suffix, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("endswith() argument must be a string")
		}
		return strings.HasSuffix(s, suffix), nil

	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("contains() takes exactly 1 argument (%d given)", len(args))
		}
		substr, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("contains() argument must be a string")
		}
		return strings.Contains(s, substr), nil

	case "lower":
		if len(args) != 0 {
			return nil, fmt.Errorf("lower() takes no arguments (%d given)", len(args))
		}
		return strings.ToLower(s), nil

	case "upper":
		if len(args) != 0 {
			return nil, fmt.Errorf("upper() takes no arguments (%d given)", len(args))
		}
		return strings.ToUpper(s), nil

	case "strip", "trim":
		if len(args) != 0 {
			return nil, fmt.Errorf("%s() takes no arguments (%d given)", method, len(args))
		}
		return strings.TrimSpace(s), nil

	default:
		return nil, fmt.Errorf("unknown string method: %s", method)
	}
}

// getField gets a field value from a post
func getField(post *models.Post, name string) (interface{}, error) {
	// Handle known fields directly for better performance
	switch name {
	case "path", "Path":
		return post.Path, nil
	case "content", "Content":
		return post.Content, nil
	case "slug", "Slug":
		return post.Slug, nil
	case "href", "Href":
		return post.Href, nil
	case "title", "Title":
		if post.Title == nil {
			return nil, nil
		}
		return *post.Title, nil
	case "date", "Date":
		if post.Date == nil {
			return nil, nil
		}
		return *post.Date, nil
	case "published", "Published":
		return post.Published, nil
	case "draft", "Draft":
		return post.Draft, nil
	case "skip", "Skip":
		return post.Skip, nil
	case "tags", "Tags":
		return post.Tags, nil
	case "description", "Description":
		if post.Description == nil {
			return nil, nil
		}
		return *post.Description, nil
	case "template", "Template":
		return post.Template, nil
	case "html", "HTML":
		return post.HTML, nil
	case "article_html", "ArticleHTML":
		return post.ArticleHTML, nil
	default:
		// Check in Extra map
		if val, ok := post.Extra[name]; ok {
			return val, nil
		}
		// Try lowercase
		if val, ok := post.Extra[strings.ToLower(name)]; ok {
			return val, nil
		}
		return nil, nil
	}
}

// getFieldFromValue gets a field from a reflect value
func getFieldFromValue(obj interface{}, field string) (interface{}, error) {
	if obj == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		fv := rv.FieldByName(field)
		if !fv.IsValid() {
			// Try case-insensitive match
			fv = rv.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, field)
			})
		}
		if fv.IsValid() {
			return fv.Interface(), nil
		}
		return nil, nil

	case reflect.Map:
		key := reflect.ValueOf(field)
		if key.Type().AssignableTo(rv.Type().Key()) {
			val := rv.MapIndex(key)
			if val.IsValid() {
				return val.Interface(), nil
			}
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("cannot access field '%s' on type %T", field, obj)
	}
}

// toBool converts a value to a boolean
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		case reflect.Ptr, reflect.Interface:
			return !rv.IsNil()
		default:
			return true
		}
	}
}

// compare compares two values and returns -1, 0, or 1
func compare(a, b interface{}) int {
	// Handle nil
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Convert to comparable types
	a = normalizeValue(a)
	b = normalizeValue(b)

	// Type-specific comparison
	switch av := a.(type) {
	case bool:
		return compareBool(av, b)
	case int64:
		return compareInt64(av, b)
	case float64:
		return compareFloat64(av, b)
	case string:
		return compareString(av, b)
	case time.Time:
		return compareTime(av, b)
	}

	// Fallback to string comparison
	return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
}

// compareBool compares a boolean value with another value
func compareBool(av bool, b interface{}) int {
	bv, ok := b.(bool)
	if !ok {
		return compareTypes(av, b)
	}
	if av == bv {
		return 0
	}
	if av {
		return 1
	}
	return -1
}

// compareInt64 compares an int64 value with another value
func compareInt64(av int64, b interface{}) int {
	switch bv := b.(type) {
	case int64:
		return compareOrdered(av, bv)
	case float64:
		return compareOrdered(float64(av), bv)
	}
	return compareTypes(av, b)
}

// compareFloat64 compares a float64 value with another value
func compareFloat64(av float64, b interface{}) int {
	switch bv := b.(type) {
	case float64:
		return compareOrdered(av, bv)
	case int64:
		return compareOrdered(av, float64(bv))
	}
	return compareTypes(av, b)
}

// compareString compares a string value with another value
func compareString(av string, b interface{}) int {
	bv, ok := b.(string)
	if !ok {
		return compareTypes(av, b)
	}
	return compareOrdered(av, bv)
}

// compareTime compares a time.Time value with another value
func compareTime(av time.Time, b interface{}) int {
	bv, ok := b.(time.Time)
	if !ok {
		return compareTypes(av, b)
	}
	if av.Before(bv) {
		return -1
	}
	if av.After(bv) {
		return 1
	}
	return 0
}

// compareOrdered compares two ordered values (numbers, strings)
func compareOrdered[T int64 | float64 | string](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// normalizeValue converts values to standard types for comparison
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case uint:
		return int64(val) //nolint:gosec // G115: conversion is safe for filter comparison values
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val) //nolint:gosec // G115: conversion is safe for filter comparison values
	case float32:
		return float64(val)
	case *time.Time:
		if val == nil {
			return nil
		}
		return *val
	default:
		return v
	}
}

// compareTypes compares different types by their type name
func compareTypes(a, b interface{}) int {
	ta := fmt.Sprintf("%T", a)
	tb := fmt.Sprintf("%T", b)
	return strings.Compare(ta, tb)
}
