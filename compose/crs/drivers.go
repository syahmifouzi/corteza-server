package crs

import (
	"context"

	"github.com/cortezaproject/corteza-server/compose/crs/capabilities"
	"github.com/cortezaproject/corteza-server/compose/crs/schema"
	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	// cStore is a simplified interface so we can use the store.Storer to assert a valid schema
	cStore interface {
		SearchComposeModules(ctx context.Context, f types.ModuleFilter) (types.ModuleSet, types.ModuleFilter, error)
		SearchComposeModuleFields(ctx context.Context, f types.ModuleFieldFilter) (types.ModuleFieldSet, types.ModuleFieldFilter, error)
		SearchComposeNamespaces(ctx context.Context, f types.NamespaceFilter) (types.NamespaceSet, types.NamespaceFilter, error)
	}

	driver interface {
		Capabilities() capabilities.Set
		Can(dsn string, capabilities ...capabilities.Capability) bool

		Store(ctx context.Context, dsn string) (Store, error)
	}

	Loader interface {
		More() bool
		Load(*schema.Collection, []*schema.Document) (coppied int, err error)

		// -1 means unknown
		Total() int
		Cursor() any
		// ... do we need anything else here?
	}

	Store interface {
		// DML
		CreateRecords(ctx context.Context, sch *schema.Collection, cc ...*schema.Document) error
		// @note providing a schema here would allow us to control what fields we return; would we find this useful?
		SearchRecords(ctx context.Context, sch *schema.Collection, filter any) (Loader, error)
		// Rest are same difference
		// - LookupRecord
		// - UpdateRecords
		// - DeleteRecords
		// - TruncateRecords

		// Healthcheck
		// AssertCollection checks if the given store is able to handle this Collection
		AssertCollection(context.Context, *schema.Collection, ...*schema.Collection) error

		// DDL
		// AddCollection requests the driver to support the specified collections
		AddCollection(context.Context, *schema.Collection, ...*schema.Collection) error
		// RemoveCollection requests the driver to remove support for the specified collections
		RemoveCollection(context.Context, *schema.Collection, ...*schema.Collection) error
		// AlterCollection requests the driver to alter the general collectio parameters
		AlterCollection(ctx context.Context, old *schema.Collection, new *schema.Collection) error
		// AlterCollectionField requests the driver to alter the specified field of the given collection
		AlterCollectionField(ctx context.Context, sch *schema.Collection, old schema.Field, new schema.Field, trans ...func(*schema.Collection, schema.Field, expr.TypedValue) (expr.TypedValue, bool, error)) error
	}
)
