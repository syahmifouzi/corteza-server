package rdbms

// dialect.go
//
// Generic SQL functions used by majority of RDBMS drivers

import (
	"fmt"
	"strings"

	"github.com/cortezaproject/corteza-server/compose/service/crs"
	"github.com/doug-martin/goqu/v9/exp"
)

// DeepIdentJSON constructs expression with chain of JSON operators
// that point value inside JSON document
//
// Supported in databases:
//
// PostgreSQL
// https://www.postgresql.org/docs/9.3/functions-json.html
//
// MySQL
// https://dev.mysql.com/doc/expman/5.7/en/json-function-experence.html
//
// SQLite
// https://www.sqlite.org/json1.html#jptr
func DeepIdentJSON(ident string, pp ...interface{}) exp.Expression {
	var (
		sql  = "?" + strings.Repeat("->?", len(pp))
		args = []interface{}{exp.ParseIdentifier(ident)}
	)

	for _, p := range pp {
		args = append(args, exp.NewLiteralExpression("?", p))
	}

	return exp.NewLiteralExpression(sql, args...)
}

// JsonPath constructs json-path string from the slice of path parts.
//
func JsonPath(pp ...interface{}) string {
	var (
		path = "$"
	)

	for i := range pp {
		switch part := pp[i].(type) {
		case string:
			path = path + "." + part
		case int:
			path = path + fmt.Sprintf("[%d]", pp[i])
		default:
			panic(fmt.Errorf("JsonPath expect string or int, got %T", i))
		}
	}

	return path
}

// SafeCast returns CASE/WHEN/THEN/ELSE expression that verifies
func SafeCast(expr exp.Expression, utype interface{}) exp.Expression {
	var (
		NULL = exp.NewLiteralExpression("NULL")
	)

	switch known := utype.(type) {
	case crs.TypeTimestamp, crs.TypeTime, crs.TypeDate:
		return CheckISO8601(expr, Cast(expr, known), NULL)
	case crs.TypeNumber:
		return CheckNumeric(expr, Cast(expr, known), NULL)
	default:
		Cast(expr, known)
	}

	panic("I do not know how to cast this expression?")
}

func Cast(expr exp.Expression, utype interface{}) exp.Expression {
	switch known := utype.(type) {
	case crs.TypeID:
		_ = known.GeneratedByStore
		return exp.NewCastExpression(expr, fmt.Sprintf("BIGINT"))
	case crs.TypeTimestamp:
		return exp.NewCastExpression(expr, fmt.Sprintf("TIMESTAMPTZ(%d)", known.Precision))
	case crs.TypeTime:
		return exp.NewCastExpression(expr, fmt.Sprintf("TIMETZ(%d)", known.Precision))
	case crs.TypeDate:
		return exp.NewCastExpression(expr, fmt.Sprintf("DATE"))
	case crs.TypeNumber:
		return exp.NewCastExpression(expr, fmt.Sprintf("NUMERIC(%d, %d)", known.Precision, known.Scale))
	case crs.TypeText:
		if known.Length > 0 {
			return exp.NewCastExpression(expr, fmt.Sprintf("VARCHAR(%d)", known.Length))
		} else {
			return exp.NewCastExpression(expr, "TEXT")
		}
	case crs.TypeJSON:
		return exp.NewCastExpression(expr, fmt.Sprintf("JSONB"))
	}

	panic("I do not know how to cast this expression?")
}

func CheckISO8601(check, casted, def exp.Expression) exp.Expression {
	var (
		cond = exp.NewBooleanExpression(
			exp.RegexpLikeOp,
			check,
			`^\d+-\d{2}-\d{2}([T\s]\d{1,2}:\d{1,2}:(\d{2}(?:\.\d*)?)(([+-](\d{2}):?(\d{2})|Z)?))$`,
		)
	)

	return exp.NewCaseExpression().When(cond, casted).Else(def)

}

func CheckNumeric(check, casted, def exp.Expression) exp.Expression {
	var (
		cond = exp.NewBooleanExpression(
			exp.RegexpLikeOp,
			check,
			`^+-\d+(\.\d+)?$`,
		)
	)

	return exp.NewCaseExpression().When(cond, casted).Else(def)

}
