package data

import (
	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	// Model
	Model struct {
		Ident      string
		Attributes []*Attribute
	}

	// naming TBD
	Attribute struct {
		Ident      string
		MultiValue bool
		// how are values encoded
		Store StoreStrategy
		// how values are converted
		Type typeCodec
	}

	typeCodec any

	// TypeID handles ID (uint64) coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeID struct {
		// @todo need to figure out how to support when IDs
		//       generated/provided by store (SERIAL/AUTOINCREMENT)
		GeneratedByStore bool
	}

	// TypeRef handles ID (uint64) coding + reference info
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeRef struct {
		RefModel     *Model
		RefAttribute *Attribute
	}

	// TypeTimestamp handles timestamp coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeTimestamp struct{ Precision uint }

	// TypeTime handles time coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeTime struct{ Precision uint }

	// TypeDate handles date coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeDate struct{}

	// TypeNumber handles number coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeNumber struct {
		Precision uint
		Scale     uint
	}

	// TypeText handles string coding
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeText struct{ Length uint }

	// TypeBoolean
	TypeBoolean struct{}

	// TypeEnum
	TypeEnum struct {
		Values []string
	}

	// TypeGeometry
	TypeGeometry struct{}

	// TypeJSON handles coding of arbitrary data into JSON structure
	// NOT TO BE CONFUSED with encodedField
	//
	// Encoding/decoding might be different depending on
	//  1) underlying store (and dialect)
	//  2) value codec (raw, json ...)
	TypeJSON struct{}

	// TypeBlob store/return data as
	TypeBlob struct{}

	TypeUUID struct{}

	// probably somewhere else (on the db level?
	getter interface {
		GetValue(name string, pos int) (expr.TypedValue, error)
	}

	setter interface {
		SetValue(name string, pos int, value expr.TypedValue) error
	}
)

//func Conv(m *types.Module) []*Field {
//	var (
//		out = make([]*Field, len(m.Fields))
//	)
//
//	for _, mf := range m.Fields {
//		f := &Field{}
//
//		switch strings.ToLower(mf.Kind) {
//		case "bool":
//			f.Type = TypeBoolean{}
//		case "datetime":
//			// @todo handle date, time only...
//			f.Type = TypeTimestamp{}
//		case "email":
//			f.Type = TypeText{}
//		case "file":
//			f.Type = TypeID{}
//		case "number":
//			f.Type = TypeNumber{Precision: mf.Options.Precision()}
//		case "record":
//			f.Type = TypeID{}
//		case "select":
//			f.Type = TypeText{}
//		case "string":
//			f.Type = TypeText{}
//		case "url":
//			f.Type = TypeText{}
//		case "user":
//			f.Type = TypeID{}
//		}
//
//		out = append(out, f)
//	}
//
//	return out
//}

//func (f field) Ident() string               { return f.ident }
//func (f field) TypeCodec() typeCodec        { return f.typeCodec }
//func (f encodedField) Ident() string        { return f.field }
//func (f encodedField) EncodedField() string { return f.field }
//func (f encodedField) TypeCodec() typeCodec { return f.typeCodec }

//// PhysicalColumn returns representation of a record field as physical column
//func PhysicalColumn(ident string, typeCodec typeCodec) *field {
//	return &field{ident: ident, typeCodec: typeCodec}
//}
//
//// JsonObjectProperty returns representation of a record field as json object property
//func JsonObjectProperty(ident, field string, typeCodec typeCodec) *encodedField {
//	return &encodedField{ident: ident, field: field, typeCodec: typeCodec}
//}
