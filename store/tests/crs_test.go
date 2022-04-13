package tests

// This file is auto-generated.
//
// Changes to this file may cause incorrect behavior and will be lost if
// the code is regenerated.
//

import (
	"context"
	"testing"

	"github.com/cortezaproject/corteza-server/compose/service/crs"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/drivers/postgres"
	"github.com/davecgh/go-spew/spew"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/stretchr/testify/require"
)

func Test_CRS(t *testing.T) {
	var (
		ctx    = context.Background()
		s, err = postgres.Connect(ctx, "postgres://darh@localhost:5432/corteza_2022_3?sslmode=disable&")
		req    = require.New(t)

		//rr types.RecordSet

		d = goqu.Dialect("postgres")

		rValue = func(f string, pos uint) exp.Expression {
			return rdbms.DeepIdentJSON("values", f, pos)
		}
	)

	req.NoError(err)

	//rr, _, err = store.SearchComposeRecords(ctx, s, nil, types.RecordFilter{
	//	ModuleID:    0,
	//	NamespaceID: 0,
	//	Query:       "",
	//	LabeledIDs:  nil,
	//	Labels:      nil,
	//	Deleted:     0,
	//	Check:       nil,
	//	Sorting:     filter.Sorting{},
	//	Paging:      filter.Paging{},
	//})
	//
	//req.NoError(err)
	//spew.Dump(rr)

	_ = s

	rd := d.Select(
		exp.ParseIdentifier("id"),
		exp.ParseIdentifier("rel_namespace"),
		exp.ParseIdentifier("rel_module"),
		exp.ParseIdentifier("created_at"),
		exp.ParseIdentifier("updated_at"),
		// ...
		exp.NewAliasExpression(rdbms.Cast(rValue("field1", 0), crs.TypeText{}), "values_field1"),
		rdbms.Cast(rValue("field2", 0), crs.TypeText{}),
	).
		From("compose_records").
		Where()

	sql, args, err := rd.ToSQL()
	println(sql)
	spew.Dump(args)
	spew.Dump(err)

	//aux := rdbms.RecordStoreCodecRDBMS(
	//	goqu.Dialect("postgres"), "foo",
	//
	//	&crs.Field{Name: "id", Type: &crs.TypeID{}},
	//	&crs.Field{Name: "namespace_id", Type: &crs.TypeID{}, Store: crs.StoreCodecAlias{Ident: "rel_namespace"}},
	//	&crs.Field{Name: "field1", Type: &crs.TypeID{}, Store: crs.StoreCodecEmbedded{Ident: "values", Path: []string{"field1"}}},
	//	//crs.PhysicalColumn("id", new(crs.TypeID)),
	//	//crs.PhysicalColumn("rel_namespace", new(crs.TypeID)),
	//	//crs.PhysicalColumn("module_id", new(crs.TypeID)),
	//	//crs.PhysicalColumn("rel_owned", new(crs.TypeID)),
	//	//crs.PhysicalColumn("created_by", new(crs.TypeID)),
	//	//crs.PhysicalColumn("created_at", new(crs.TypeTimestamp)),
	//	//crs.PhysicalColumn("updated_by", new(crs.TypeID)),
	//	//crs.PhysicalColumn("updated_at", new(crs.TypeTimestamp)),
	//	//crs.PhysicalColumn("deleted_by", new(crs.TypeID)),
	//	//crs.PhysicalColumn("deleted_at", new(crs.TypeTimestamp)),
	//	//crs.JsonObjectProperty("values", "foo", new(crs.TypeText)),
	//	//crs.JsonObjectProperty("valuesExtra", "foo2", new(crs.TypeText)),
	//	//crs.JsonObjectProperty("valuesExtra", "someRecordRef", new(crs.TypeID)),
	//)
	//
	//rr, _, err = aux.Select(types.RecordFilter{})
	//
	//req.NoError(err)
	//spew.Dump(rr)
}
