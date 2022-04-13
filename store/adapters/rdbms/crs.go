package rdbms

//import (
//	"github.com/cortezaproject/corteza-server/compose/service/crs"
//	"github.com/cortezaproject/corteza-server/compose/types"
//	"github.com/davecgh/go-spew/spew"
//	"github.com/doug-martin/goqu/v9"
//	"github.com/doug-martin/goqu/v9/exp"
//)
//
//type (
//	sqlColumnCodec interface {
//		Ident() string
//	}
//
//	rawValue interface {
//		sqlColumnCodec
//		IsRaw()
//	}
//
//	encodedValue interface {
//		sqlColumnCodec
//		EncodedField() string
//	}
//
//	recordStoreCodecRDBMS struct {
//		dialect goqu.DialectWrapper
//
//		table string
//
//		// all recognised fields and how we convert them to/from RDBMS table
//		fields []*crs.Field
//	}
//
//	// TBD - to be defined; placeholder
//	TBD interface{}
//)
//
//// RecordStoreCodecRDBMS creates new a new coded for encoding (storing) and decoding (reading)
//// records to a RDBMS backend
//func RecordStoreCodecRDBMS(dialect goqu.DialectWrapper, table string, ff ...*crs.Field) *recordStoreCodecRDBMS {
//	a := &recordStoreCodecRDBMS{
//		dialect: dialect,
//		table:   table,
//		fields:  ff,
//	}
//
//	return a
//}
//
//func (c *recordStoreCodecRDBMS) Select(f types.RecordFilter) (types.RecordSet, types.RecordFilter, error) {
//	sd := c.dialect.Select(
//		// all selectable columns here...
//		// we need to specify all raw value columns but can merge
//		// all columns with (json)encoded value columns into one (depending on different idents)
//		func() (columns []interface{}) {
//			for _, field := range c.fields {
//				var col exp.Expression
//
//				switch strategy := field.Store.(type) {
//				case nil:
//					col = exp.ParseIdentifier(field.Name)
//
//				case *crs.StoreCodecAlias:
//					// using column directly
//					// @todo should we use safe casting here?
//					col = exp.ParseIdentifier(strategy.Ident)
//
//				case *crs.StoreCodecEmbedded:
//					// using JSON to handle embedded values
//					col = DeepIdentJSON(strategy.Ident, strategy.Path...)
//				}
//
//				columns = append(columns, col)
//			}
//
//			return
//		}()...,
//	).
//		From(goqu.T(c.table))
//
//	spew.Dump(sd.ToSQL())
//
//	return nil, f, nil
//}
//
//// Insert inserts one or more records
////
//// Question: should CRS handle constraint checking same as DB does or should that be handled by record service
//func (c *recordStoreCodecRDBMS) Insert(record ...*types.Record) error {
//	// @todo implement...
//	return nil
//}
//
//// Update updates one or more record
////
//// Question: should CRS handle constraint checking same as DB does or should that be handled by record service
//func (c *recordStoreCodecRDBMS) Update(record ...*types.Record) error {
//	// @todo implement...
//	return nil
//}
//
//// PartialUpdate updates all matching records (filter) with the given expression
////
//// Not sure how we would even implement this? Where should the constraint checking be done?
//func (c *recordStoreCodecRDBMS) PartialUpdate(expr TBD, f types.RecordFilter) error {
//	// @todo implement...
//	return nil
//}
//
////
//func (c *recordStoreCodecRDBMS) DeleteByID(ii ...uint64) error {
//	// @todo implement...
//	return nil
//}
//
//// Just an idea...
//func (c *recordStoreCodecRDBMS) BulkDelete(f types.RecordFilter) error {
//	// @todo implement...
//	return nil
//}

// showcasing construction of new record store codec for a specific module & its fields.
//func poc(cfg *ConnConfig) {
//	crsCodecForMyPrettyModule := RecordStoreCodecRDBMS(
//		cfg.Dialect,
//		"myPrettyModule",
//		crs.PhysicalColumn("id", new(crs.TypeUUID)),
//		crs.PhysicalColumn("rel_namespace", new(crs.TypeID)),
//		crs.PhysicalColumn("module_id", new(crs.TypeID)),
//		crs.PhysicalColumn("rel_owned", new(crs.TypeID)),
//		crs.PhysicalColumn("created_by", new(crs.TypeID)),
//		crs.PhysicalColumn("created_at", new(crs.TypeTimestamp)),
//		crs.PhysicalColumn("updated_by", new(crs.TypeID)),
//		crs.PhysicalColumn("updated_at", new(crs.TypeTimestamp)),
//		crs.PhysicalColumn("deleted_by", new(crs.TypeID)),
//		crs.PhysicalColumn("deleted_at", new(crs.TypeTimestamp)),
//		crs.JsonObjectProperty("values", "foo", new(crs.TypeText)),
//		crs.JsonObjectProperty("valuesExtra", "foo2", new(crs.TypeText)),
//		crs.JsonObjectProperty("valuesExtra", "someRecordRef", new(crs.TypeID)),
//		crs.PhysicalColumn("deleted_At", new(crs.TypeTimestamp)),
//	)
//
//	_ = crsCodecForMyPrettyModule
//}
