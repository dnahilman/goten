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

// TxRunner is an optional capability: adapters that support database transactions
// implement it. fn runs with a context carrying the active transaction; all
// Adapter calls made with that context participate in the same transaction, and
// returning a non-nil error (or panicking) rolls it back. Adapters that don't
// implement TxRunner simply run without transactional guarantees.
type TxRunner interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

func EQ(field string, value any) Where {
	return Where{Field: field, Operator: "=", Value: value}
}
