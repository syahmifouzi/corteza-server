package schema

import (
	"fmt"

	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	FieldOption func(*field)
	fieldType   string
	Field       interface {
		Ident() string
		Label() string
		Type() fieldType
		ToTyped(v any) (expr.TypedValue, error)
		ToRV(v expr.TypedValue) (string, error)
		IsMultiValue() bool
	}

	field struct {
		ident      string
		label      string
		fieldType  fieldType
		multiValue bool
	}

	fieldString struct {
		field
	}

	fieldDate struct {
		field
	}

	fieldRef struct {
		field
		RefResource string
	}

	fieldNumber struct {
		field
		Precision uint
	}
)

const (
	FieldTypeString fieldType = "string"
	FieldTypeDate   fieldType = "date"
	FieldTypeRef    fieldType = "ref"
	FieldTypeNumber fieldType = "number"
)

// Base field def

func (f field) Ident() string {
	return f.ident
}
func (f field) Label() string {
	return f.label
}
func (f field) Type() fieldType {
	return f.fieldType
}
func (f field) IsMultiValue() bool {
	return f.multiValue
}

func prepareField(ident, label string, t fieldType, opts ...FieldOption) field {
	f := &field{
		ident:     ident,
		label:     label,
		fieldType: t,
	}

	for _, o := range opts {
		o(f)
	}
	return *f
}

// Field stuff

func FieldString(ident, label string, opts ...FieldOption) fieldString {
	f := fieldString{
		field: prepareField(ident, label, FieldTypeString, opts...),
	}
	return f
}
func (f fieldString) ToTyped(v any) (expr.TypedValue, error) {
	return expr.NewString(v)
}
func (f fieldString) ToRV(v expr.TypedValue) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func FieldDate(ident, label string, opts ...FieldOption) fieldDate {
	f := fieldDate{
		field: prepareField(ident, label, FieldTypeDate, opts...),
	}
	return f
}
func (f fieldDate) ToTyped(v any) (expr.TypedValue, error) {
	return expr.NewDateTime(v)
}
func (f fieldDate) ToRV(v expr.TypedValue) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func FieldRef(ident, label string, refResource string, opts ...FieldOption) fieldRef {
	f := fieldRef{
		field: prepareField(ident, label, FieldTypeRef, opts...),
	}
	f.RefResource = refResource
	return f
}
func FieldRefGeneric(ident, label string, opts ...FieldOption) fieldRef {
	f := fieldRef{
		field: prepareField(ident, label, FieldTypeRef, opts...),
	}
	return f
}
func (f fieldRef) ToTyped(v any) (expr.TypedValue, error) {
	return expr.NewID(v)
}
func (f fieldRef) ToRV(v expr.TypedValue) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func FieldNumber(ident, label string, opts ...FieldOption) fieldNumber {
	f := fieldNumber{
		field: prepareField(ident, label, FieldTypeNumber, opts...),
	}
	return f
}
func (f fieldNumber) ToTyped(v any) (expr.TypedValue, error) {
	if f.Precision != 0 {
		return expr.NewFloat(v)
	}
	return expr.NewInteger(v)
}
func (f fieldNumber) ToRV(v expr.TypedValue) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// Functional options

func WithMultipleValues() FieldOption {
	return func(f *field) {
		f.multiValue = true
	}
}
