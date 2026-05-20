package testutil

import (
	"context"
	"sync"

	adp "github.com/dnahilman/goten/adapter"
)

// MockAdapter is an in-memory Adapter for unit tests.
type MockAdapter struct {
	mu   sync.RWMutex
	data map[string][]map[string]any
}

var _ adp.Adapter = (*MockAdapter)(nil)

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{data: make(map[string][]map[string]any)}
}

func (m *MockAdapter) FindOne(ctx context.Context, model string, q adp.Query) (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, row := range m.data[model] {
		if matchesAll(row, q.Where) {
			return copyRow(row), nil
		}
	}
	return nil, nil
}

func (m *MockAdapter) FindMany(ctx context.Context, model string, q adp.Query) ([]map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []map[string]any
	for _, row := range m.data[model] {
		if matchesAll(row, q.Where) {
			out = append(out, copyRow(row))
		}
	}
	return out, nil
}

func (m *MockAdapter) Create(ctx context.Context, model string, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := copyRow(data)
	m.data[model] = append(m.data[model], cp)
	return copyRow(cp), nil
}

func (m *MockAdapter) Update(ctx context.Context, model string, q adp.Query, data map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, row := range m.data[model] {
		if matchesAll(row, q.Where) {
			for k, v := range data {
				m.data[model][i][k] = v
			}
			return copyRow(m.data[model][i]), nil
		}
	}
	return nil, nil
}

func (m *MockAdapter) Delete(ctx context.Context, model string, q adp.Query) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []map[string]any
	for _, row := range m.data[model] {
		if !matchesAll(row, q.Where) {
			kept = append(kept, row)
		}
	}
	m.data[model] = kept
	return nil
}

func (m *MockAdapter) Count(ctx context.Context, model string, q adp.Query) (int64, error) {
	rows, err := m.FindMany(ctx, model, q)
	return int64(len(rows)), err
}

func matchesAll(row map[string]any, wheres []adp.Where) bool {
	for _, w := range wheres {
		if w.Operator == "=" && row[w.Field] != w.Value {
			return false
		}
	}
	return true
}

func copyRow(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
