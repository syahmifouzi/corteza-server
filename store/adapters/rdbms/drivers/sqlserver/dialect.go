package sqlserver

import (
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

// DeepIdentJSON constructs JSON_VALUE function call
// returns a value from JSON document using JSON path
//
// Supported in databases:
//
// SQL Server
// https://docs.microsoft.com/en-us/sql/relational-databases/json/json-path-expressions-sql-server?view=sql-server-ver15
func DeepIdentJSON(ident string, pp ...interface{}) exp.Expression {
	if len(pp) == 0 {
		return goqu.I(ident)
	}

	return goqu.Func("JSON_VALUE", goqu.I(ident), goqu.V(rdbms.JsonPath(pp...)))
}
