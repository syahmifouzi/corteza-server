package dal

import (
	"context"
	"fmt"
	"strings"

	"github.com/PaesslerAG/gval"
	"github.com/cortezaproject/corteza-server/pkg/dal/capabilities"
	"github.com/cortezaproject/corteza-server/pkg/handle"
	"github.com/modern-go/reflect2"
)

type (
	IdentFormatter struct {
		defaultModelIdent     string
		defaultAttributeIdent string

		defaultPartitionFormat string

		things map[string]any

		partitionFormatValidator gval.Evaluable
	}

	// ModelFilter is used to retrieve a model from the DAL based on given params
	ModelFilter struct {
		ConnectionID uint64

		ResourceID uint64

		ResourceType string
		Resource     string
	}

	// Model describes the underlying data and its shape
	Model struct {
		ConnectionID uint64
		Ident        string
		Label        string

		Resource     string
		ResourceID   uint64
		ResourceType string

		SensitivityLevel uint64

		Attributes AttributeSet

		Capabilities capabilities.Set
	}
	ModelSet []*Model

	// Attribute describes a specific value of the dataset
	Attribute struct {
		Ident string
		Label string

		SensitivityLevel uint64

		MultiValue bool

		PrimaryKey bool

		// If attribute has SoftDeleteFlag=true we use it
		// when filtering out deleted items
		SoftDeleteFlag bool

		// Is attribute sortable?
		// Note: all primary keys are sortable
		Sortable bool

		// Can attribute be used in query expression?
		Filterable bool

		// Store describes the strategy the underlying storage system should
		// apply to the underlying value
		Store Codec

		// Type describes what the value represents and how it should be
		// encoded/decoded
		Type Type
	}

	AttributeSet []*Attribute
)

func PrimaryAttribute(ident string, codec Codec) *Attribute {
	out := FullAttribute(ident, TypeID{}, codec)
	out.Type = &TypeID{}
	out.PrimaryKey = true
	return out
}

func FullAttribute(ident string, at Type, codec Codec) *Attribute {
	return &Attribute{
		Ident:      ident,
		Label:      ident,
		Sortable:   true,
		Filterable: true,
		Store:      codec,
		Type:       at,
	}
}

func (a *Attribute) WithSoftDelete() *Attribute {
	a.SoftDeleteFlag = true
	return a
}

func (a *Attribute) WithMultiValue() *Attribute {
	a.MultiValue = true
	return a
}

func (mm ModelSet) FindByResourceID(resourceID uint64) *Model {
	for _, m := range mm {
		if m.ResourceID == resourceID {
			return m
		}
	}
	return nil
}

func (mm ModelSet) FindByResourceIdent(resourceType, resourceIdent string) *Model {
	for _, m := range mm {
		if m.ResourceType == resourceType && m.Resource == resourceIdent {
			return m
		}
	}
	return nil
}

func (mm ModelSet) FindByIdent(ident string) *Model {
	for _, m := range mm {
		if m.Ident == ident {
			return m
		}
	}

	return nil
}

// FilterByReferenced returns all of the models that reference b
func (aa ModelSet) FilterByReferenced(b *Model) (out ModelSet) {
	for _, aModel := range aa {
		if aModel.Resource == b.Resource {
			continue
		}

		for _, aAttribute := range aModel.Attributes {
			switch casted := aAttribute.Type.(type) {
			case *TypeRef:
				if casted.RefModel.Resource == b.Resource {
					out = append(out, aModel)
				}
			}
		}
	}

	return
}

func (m Model) ToFilter() ModelFilter {
	return ModelFilter{
		ConnectionID: m.ConnectionID,

		ResourceID: m.ResourceID,

		ResourceType: m.ResourceType,
		Resource:     m.Resource,
	}
}

// HasAttribute returns true when the model includes the specified attribute
func (m Model) HasAttribute(ident string) bool {
	return m.Attributes.FindByIdent(ident) != nil
}

func (aa AttributeSet) FindByIdent(ident string) *Attribute {
	for _, a := range aa {
		if strings.EqualFold(a.Ident, ident) {
			return a
		}
	}

	return nil
}

// Validate performs a base model validation before it is passed down
func (m Model) Validate() error {
	if m.Resource == "" {
		return fmt.Errorf("resource not defined")
	}

	seen := make(map[string]bool)
	for _, attr := range m.Attributes {
		if attr.Ident == "" {
			return fmt.Errorf("invalid attribute ident: ident must not be empty")
		}

		if !handle.IsValid(attr.Ident) {
			return fmt.Errorf("invalid attribute ident: %s is not a valid handle", attr.Ident)
		}

		if seen[attr.Ident] {
			return fmt.Errorf("invalid attribute %s: duplicate attributes are not allowed", attr.Ident)
		}
		seen[attr.Ident] = true

		if reflect2.IsNil(attr.Type) {
			return fmt.Errorf("attribute does not define a type: %s", attr.Ident)
		}
	}

	return nil
}

func (f *IdentFormatter) AddEvalParam(params map[string]any) {
	if f.things == nil {
		f.things = make(map[string]any)
	}

	for k, v := range params {
		f.things[k] = v
	}

}

func (f IdentFormatter) getEvalParams(ident string) (out map[string]any) {
	out = map[string]any{
		"ident": ident,
	}

	for k, v := range f.things {
		out[k] = v
	}

	return
}

func (f IdentFormatter) ModelIdent(ctx context.Context, partitioned bool, tpl string, kv ...string) (out string, ok bool) {
	ok = true

	if !partitioned {
		return f.defaultModelIdent, ok
	}

	// A bit of preprocessing
	{
		if len(kv)%2 != 0 {
			panic("ModelIdentFormatter.ModelIdent requires key/value pairs")
		}
		for i := 0; i < len(kv); i += 2 {
			kv[i] = fmt.Sprintf("{{%s}}", kv[i])
		}
	}

	// Template preparation with defaulting based on provided KV
	{
		if tpl == "" {
			tpl = f.defaultPartitionFormat
		}
		if tpl == "" {
			pts := make([]string, 0, len(kv)/2)
			for i := 0; i < len(kv); i += 2 {
				pts = append(pts, kv[i])
			}
			tpl = strings.Join(pts, "_")
		}
		if tpl == "" {
			return "", false
		}
	}

	rpl := strings.NewReplacer(kv...)
	out = rpl.Replace(tpl)

	if f.partitionFormatValidator != nil {
		var err error
		ok, err = f.partitionFormatValidator.EvalBool(ctx, f.getEvalParams(out))
		ok = ok && (err == nil)
	}

	return
}

func (f IdentFormatter) AttributeIdent(partitioned bool, ident string) (out string, ok bool) {
	if !partitioned {
		if f.defaultAttributeIdent != "" {
			return f.defaultAttributeIdent, true
		}
		return "", false
	}

	return ident, ident != ""

}
