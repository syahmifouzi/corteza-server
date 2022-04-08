// This package is just so that I have an interface conforming driver

package testko

import (
	"context"
	"fmt"
	"strings"

	"github.com/cortezaproject/corteza-server/compose/crs"
	"github.com/cortezaproject/corteza-server/compose/crs/capabilities"
	"github.com/cortezaproject/corteza-server/compose/crs/schema"
	"github.com/cortezaproject/corteza-server/pkg/expr"
)

type (
	testkoDriver struct{}

	testkoStore map[string]schema.DocumentSet
)

var (
	gStore = make(testkoStore)
)

func Driver() *testkoDriver {
	return &testkoDriver{}
}

func (d testkoDriver) Capabilities() capabilities.Set {
	return capabilities.FullCapabilities()
}

func (d testkoDriver) Can(uri string, capabilities ...capabilities.Capability) bool {
	return strings.HasPrefix(uri, "testko://") && d.Capabilities().Can(capabilities...)
}

func (d testkoDriver) Store(ctx context.Context, uri string) (str crs.Store, err error) {
	return &testkoStore{}, nil
}

// Healthcheck

func (ds *testkoStore) AssertCollection(ctx context.Context, c *schema.Collection, cc ...*schema.Collection) error {
	return nil
}

// DML

func (ds *testkoStore) CreateRecords(ctx context.Context, sch *schema.Collection, cc ...*schema.Document) error {
	gStore[sch.Ident()] = append(gStore[sch.Ident()], cc...)
	return nil
}

func (ds *testkoStore) SearchRecords(ctx context.Context, sch *schema.Collection, filter any) (crs.Loader, error) {
	return nil, fmt.Errorf("not implemented")
}

// DDL

func (ds *testkoStore) AddCollection(context.Context, *schema.Collection, ...*schema.Collection) error {
	return fmt.Errorf("not implemented")
}

func (ds *testkoStore) RemoveCollection(context.Context, *schema.Collection, ...*schema.Collection) error {
	return fmt.Errorf("not implemented")
}

func (ds *testkoStore) AlterCollection(ctx context.Context, old *schema.Collection, new *schema.Collection) error {
	return fmt.Errorf("not implemented")
}

func (ds *testkoStore) AlterCollectionField(ctx context.Context, sch *schema.Collection, old schema.Field, new schema.Field, trans ...func(*schema.Collection, schema.Field, expr.TypedValue) (expr.TypedValue, bool, error)) error {
	return fmt.Errorf("not implemented")

}
