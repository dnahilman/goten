// Package adapter defines the database adapter interface used by goten core and plugins.
package adapter

import "context"

type Query struct {
	Where   []Where
	Limit   int
	Offset  int
	SortBy  string
	SortDir string
}

type Where struct {
	Field    string
	Operator string // "=", "!=", ">", "<", ">=", "<=", "in", "like"
	Value    any
}

// Adapter is the database abstraction all goten components depend on.
type Adapter interface {
	FindOne(ctx context.Context, model string, q Query) (map[string]any, error)
	FindMany(ctx context.Context, model string, q Query) ([]map[string]any, error)
	Create(ctx context.Context, model string, data map[string]any) (map[string]any, error)
	Update(ctx context.Context, model string, q Query, data map[string]any) (map[string]any, error)
	Delete(ctx context.Context, model string, q Query) error
	Count(ctx context.Context, model string, q Query) (int64, error)
}

func EQ(field string, value any) Where {
	return Where{Field: field, Operator: "=", Value: value}
}
