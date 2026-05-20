package goten_test

import (
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/stretchr/testify/assert"
)

func TestEQ(t *testing.T) {
	w := goten.EQ("email", "a@b.com")
	assert.Equal(t, "email", w.Field)
	assert.Equal(t, "=", w.Operator)
	assert.Equal(t, "a@b.com", w.Value)
}

func TestQuery_Defaults(t *testing.T) {
	q := goten.Query{}
	assert.Nil(t, q.Where)
	assert.Equal(t, 0, q.Limit)
	assert.Equal(t, 0, q.Offset)
}

func TestQuery_WithMultipleWheres(t *testing.T) {
	q := goten.Query{
		Where: []goten.Where{
			goten.EQ("id", "g10_abc"),
			goten.EQ("email_verified", true),
		},
	}
	assert.Len(t, q.Where, 2)
	assert.Equal(t, "id", q.Where[0].Field)
	assert.Equal(t, "email_verified", q.Where[1].Field)
}
