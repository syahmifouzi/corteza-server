package schema

import (
	"fmt"

	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	DocumentBuilder func() (k string, v interface{})

	Collection struct {
		ident string
		label string

		fields []Field
	}

	documentValues map[string][]expr.TypedValue
	Document       struct {
		errors []error
		values documentValues
	}
	DocumentSet []*Document
)

func NewCollection(ident, label string, ff ...Field) (*Collection, error) {
	c := &Collection{
		ident: ident,
		label: label,
	}

	return c.addField(ff...)
}

func (s *Collection) AddField(f Field, fields ...Field) (*Collection, error) {
	return s.addField(append([]Field{f}, fields...)...)
}

func (c *Collection) addField(fields ...Field) (*Collection, error) {
	for _, f := range fields {
		if c.HasField(f.Ident()) {
			return c, fmt.Errorf("duplicate field %s in schema %s", f.Ident(), c.ident)
		}

		c.fields = append(c.fields, f)
	}

	return c, nil
}

func (s Collection) Field(k string) (out Field) {
	for _, f := range s.fields {
		if f.Ident() == k {
			return f
		}
	}

	return
}

func (s *Collection) HasField(k string) bool {
	for _, f := range s.fields {
		if f.Ident() == k {
			return true
		}
	}

	return false
}

func (c Collection) Ident() string {
	return c.ident
}

func (s Collection) BuildDocument(nn ...DocumentBuilder) (d *Document, err error) {
	d = &Document{
		values: make(documentValues),
	}

	for _, n := range nn {
		k, vRaw := n()

		f := s.Field(k)
		if f == nil {
			return nil, fmt.Errorf("schema %s does not define field %s", s.ident, k)
		}

		d, err = s.addDocumentValue(d, f, k, vRaw)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (s Collection) AddDocumentValue(d *Document, k string, vRaw any) *Document {
	f := s.Field(k)
	if f == nil {
		d.errors = append(d.errors, fmt.Errorf("schema %s does not define field %s", s.ident, k))
		return d
	}

	d, err := s.addDocumentValue(d, f, k, vRaw)
	if err != nil {
		d.errors = append(d.errors, err)
		return d
	}

	return d
}

func (s Collection) addDocumentValue(d *Document, f Field, k string, vRaw any) (*Document, error) {
	v, err := f.ToTyped(vRaw)
	if err != nil {
		return nil, err
	}

	if _, ok := d.values[k]; ok && !f.IsMultiValue() {
		return nil, fmt.Errorf("field %s is not a multi value field and a value already exists", f.Ident())
	}

	d.values[k] = append(d.values[k], v)

	return d, nil
}

func (s Collection) GetValue(d *Document, k string) ([]expr.TypedValue, error) {
	// @todo make something nicer
	if len(d.errors) > 0 {
		return nil, d.errors[0]
	}

	f := s.Field(k)
	if f == nil {
		return nil, fmt.Errorf("schema %s does not define field %s", s.ident, k)
	}

	return s.getValue(d, k), nil
}

func (c Collection) WalkValues(d *Document, f func(k string, f Field, v []expr.TypedValue) error) (err error) {
	for _, field := range c.fields {
		err = f(field.Ident(), field, c.getValue(d, field.Ident()))
		if err != nil {
			return
		}
	}

	return
}

func (s Collection) GetValueAt(d *Document, k string, index int) (expr.TypedValue, error) {
	// @todo make something nicer
	if len(d.errors) > 0 {
		return nil, d.errors[0]
	}

	f := s.Field(k)
	if f == nil {
		return nil, fmt.Errorf("schema %s does not define field %s", s.ident, k)
	}

	vv := s.getValue(d, k)
	if len(vv)-1 < index {
		return nil, fmt.Errorf("cannot access value at index %d: %d values available", index, len(vv))
	}

	return s.getValue(d, k)[index], nil
}

func (s Collection) getValue(pl *Document, k string) []expr.TypedValue {
	return pl.values[k]
}
