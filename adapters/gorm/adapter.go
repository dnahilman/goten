package gormadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	adp "github.com/dnahilman/goten/adapter"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Adapter implements adp.Adapter using GORM.
type Adapter struct {
	db *gorm.DB
}

// compile-time interface check
var _ adp.Adapter = (*Adapter)(nil)

// New creates a new GORM adapter with silent logger.
func New(db *gorm.DB) *Adapter {
	return &Adapter{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}

var validOperators = map[string]bool{
	"=": true, "!=": true, ">": true, "<": true,
	">=": true, "<=": true, "in": true, "like": true,
}

func isValidOperator(op string) bool { return validOperators[strings.ToLower(op)] }

// quoteIdent wraps a column name in double-quotes (Postgres standard).
func quoteIdent(name string) (string, error) {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return "", fmt.Errorf("invalid identifier: %q", name)
		}
	}
	return `"` + name + `"`, nil
}

func (a *Adapter) applyWheres(tx *gorm.DB, wheres []adp.Where) (*gorm.DB, error) {
	for _, w := range wheres {
		op := strings.ToLower(w.Operator)
		if !isValidOperator(op) {
			return nil, fmt.Errorf("invalid operator: %q", w.Operator)
		}
		col, err := quoteIdent(w.Field)
		if err != nil {
			return nil, err
		}
		if op == "in" {
			tx = tx.Where(col+" IN ?", w.Value)
		} else {
			tx = tx.Where(col+" "+op+" ?", w.Value)
		}
	}
	return tx, nil
}

func (a *Adapter) FindOne(ctx context.Context, model string, q adp.Query) (map[string]any, error) {
	tx := a.db.WithContext(ctx).Table(model)
	var err error
	if tx, err = a.applyWheres(tx, q.Where); err != nil {
		return nil, err
	}
	if q.SortBy != "" {
		col, err := quoteIdent(q.SortBy)
		if err != nil {
			return nil, err
		}
		dir := "ASC"
		if strings.ToLower(q.SortDir) == "desc" {
			dir = "DESC"
		}
		tx = tx.Order(col + " " + dir)
	}
	var result map[string]any
	if err := tx.Take(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (a *Adapter) FindMany(ctx context.Context, model string, q adp.Query) ([]map[string]any, error) {
	tx := a.db.WithContext(ctx).Table(model)
	var err error
	if tx, err = a.applyWheres(tx, q.Where); err != nil {
		return nil, err
	}
	if q.SortBy != "" {
		col, err := quoteIdent(q.SortBy)
		if err != nil {
			return nil, err
		}
		dir := "ASC"
		if strings.ToLower(q.SortDir) == "desc" {
			dir = "DESC"
		}
		tx = tx.Order(col + " " + dir)
	}
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}
	var results []map[string]any
	if err := tx.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (a *Adapter) Create(ctx context.Context, model string, data map[string]any) (map[string]any, error) {
	if err := a.db.WithContext(ctx).Table(model).Create(&data).Error; err != nil {
		return nil, err
	}
	return data, nil
}

func (a *Adapter) Update(ctx context.Context, model string, q adp.Query, data map[string]any) (map[string]any, error) {
	tx := a.db.WithContext(ctx).Table(model)
	var err error
	if tx, err = a.applyWheres(tx, q.Where); err != nil {
		return nil, err
	}
	if err := tx.Select(keysOf(data)).Updates(data).Error; err != nil {
		return nil, err
	}
	return a.FindOne(ctx, model, q)
}

func (a *Adapter) Delete(ctx context.Context, model string, q adp.Query) error {
	tx := a.db.WithContext(ctx).Table(model)
	var err error
	if tx, err = a.applyWheres(tx, q.Where); err != nil {
		return err
	}
	return tx.Delete(nil).Error
}

func (a *Adapter) Count(ctx context.Context, model string, q adp.Query) (int64, error) {
	tx := a.db.WithContext(ctx).Table(model)
	var err error
	if tx, err = a.applyWheres(tx, q.Where); err != nil {
		return 0, err
	}
	var count int64
	return count, tx.Count(&count).Error
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
