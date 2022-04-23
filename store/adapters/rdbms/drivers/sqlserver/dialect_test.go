package sqlserver

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"

	_ "github.com/doug-martin/goqu/v9/dialect/sqlserver"
)

// test deep ident expression generator
func Test_DeepIdentJSON(t *testing.T) {
	var (
		pre  = `SELECT `
		post = ` FROM "test"`

		cc = []struct {
			input []interface{}
			sql   string
			args  []interface{}
		}{
			{
				input: []interface{}{"one"},
				sql:   `"one"`,
				args:  []interface{}{},
			},
			{
				input: []interface{}{"one", "two"},
				sql:   `JSON_VALUE("one", '$.two')`,
				args:  []interface{}{},
			},
		}
	)

	for _, c := range cc {
		t.Run(c.sql, func(t *testing.T) {
			var (
				r = require.New(t)
			)

			sql, args, err := goqu.Dialect("sqlserver").Select(DeepIdentJSON(c.input[0].(string), c.input[1:]...)).From("test").ToSQL()
			r.NoError(err)
			r.Equal(pre+c.sql+post, sql)
			r.Equal(c.args, args)
		})
	}
}
