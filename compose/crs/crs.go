package crs

import (
	"context"
	"fmt"

	"github.com/cortezaproject/corteza-server/compose/crs/capabilities"
	"github.com/cortezaproject/corteza-server/compose/crs/schema"
	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/expr"
	systemTypes "github.com/cortezaproject/corteza-server/system/types"
)

type (
	// @todo perhaps an aux struct would be better here?
	crsDefiner interface {
		ComposeRecordStoreID() uint64
		StoreDSN() string
		Capabilities() capabilities.Set
	}

	composeRecordStore struct {
		drivers []driver
		stores  map[uint64]*storeWrap

		primary *storeWrap
	}

	storeWrap struct {
		store        connFunc
		capabilities capabilities.Set
	}

	connFunc func(ctx context.Context) (Store, error)
)

func ComposeRecordStore(ctx context.Context, primary crsDefiner, drivers ...driver) (*composeRecordStore, error) {
	crs := &composeRecordStore{
		drivers: drivers,
		stores:  make(map[uint64]*storeWrap),
		primary: nil,
	}

	d, err := crs.getSupportingDriver(primary)
	if err != nil {
		return nil, err
	}

	crs.primary = &storeWrap{
		store:        storeConnWrap(d, primary),
		capabilities: primary.Capabilities(),
	}

	return crs, nil
}

func (crs *composeRecordStore) AddStore(ctx context.Context, definers ...crsDefiner) error {
	for _, definer := range definers {
		d, err := crs.getSupportingDriver(definer)
		if err != nil {
			return err
		}
		crs.stores[definer.ComposeRecordStoreID()] = &storeWrap{
			store:        storeConnWrap(d, definer),
			capabilities: definer.Capabilities(),
		}
	}
	return nil
}

func (crs *composeRecordStore) RemoveStore(ctx context.Context, definers ...crsDefiner) error {
	for _, definer := range definers {
		// @todo probably some cleanup and stuff
		delete(crs.stores, definer.ComposeRecordStoreID())
	}

	return nil
}

func (crs *composeRecordStore) getStore(ctx context.Context, storeID uint64, cc ...capabilities.Capability) (store Store, can capabilities.Set, err error) {
	// Get existence
	var wrap *storeWrap
	if storeID == 0 {
		wrap = crs.primary
	} else {
		wrap = crs.stores[storeID]
	}
	if wrap == nil {
		err = fmt.Errorf("crs store does not exist: %d", storeID)
		return
	}

	// Check capabilities
	if !wrap.capabilities.Can(cc...) {
		err = fmt.Errorf("crs store does not support requested capabilities: %d", storeID)
		return
	}

	store, err = wrap.store(ctx)
	can = wrap.capabilities
	return
}

// ---

func (crs *composeRecordStore) getModuleSchema(mod *types.Module) (out *schema.Collection, err error) {
	ff := make([]schema.Field, len(mod.Fields))

	for i, f := range mod.Fields {
		var opts []schema.FieldOption
		switch {
		case f.Multi:
			opts = append(opts, schema.WithMultipleValues())
		}

		switch f.Kind {
		case "String":
			ff[i] = schema.FieldString(f.Name, f.Label, opts...)
		case "Email":
			ff[i] = schema.FieldString(f.Name, f.Label, opts...)
		case "Record":
			ff[i] = schema.FieldRef(f.Name, f.Label, types.RecordResourceType, opts...)
		case "User":
			ff[i] = schema.FieldRef(f.Name, f.Label, systemTypes.UserResourceType, opts...)
		case "File":
			ff[i] = schema.FieldRefGeneric(f.Name, f.Label, opts...)
		default:
			panic(fmt.Sprintf("field kind no supported: %s %s", f.Name, f.Kind))

			// @todo...
		}
	}

	ff = append(ff,
		schema.FieldRef("recordID", "Record ID", types.RecordResourceType),
		schema.FieldDate("createdAt", "Created At"),
		schema.FieldDate("updatedAt", "Created At"),
		schema.FieldDate("deletedAt", "Created At"),
		schema.FieldRef("ownedBy", "Owned By", systemTypes.UserResourceType),
		schema.FieldRef("createdBy", "Created By", systemTypes.UserResourceType),
		schema.FieldRef("updatedBy", "Updated By", systemTypes.UserResourceType),
		schema.FieldRef("deletedBy", "Deleted By", systemTypes.UserResourceType),
	)

	out, err = schema.NewCollection(
		mod.Handle,
		mod.Name,
		ff...,
	)

	return
}

func wrap(k string, v interface{}) schema.DocumentBuilder {
	return func() (string, interface{}) {
		return k, v
	}
}

func (crs *composeRecordStore) getSupportingDriver(def crsDefiner) (driver, error) {
	for _, d := range crs.drivers {
		if !d.Can(def.StoreDSN(), def.Capabilities()...) {
			continue
		}

		return d, nil
	}

	return nil, fmt.Errorf("unable to add compose record store: no driver supports this one")
}

func storeConnWrap(d driver, def crsDefiner) connFunc {
	return func(ctx context.Context) (Store, error) {
		return d.Store(ctx, def.StoreDSN())
	}
}

func (crs *composeRecordStore) recordToCollection(sch *schema.Collection, records ...*types.Record) (out []*schema.Document, err error) {
	for _, r := range records {

		builders := []schema.DocumentBuilder{
			wrap("recordID", r.ID),
			wrap("createdAt", r.CreatedAt),
			wrap("updatedAt", r.UpdatedAt),
			wrap("deletedAt", r.DeletedAt),
			wrap("ownedBy", r.OwnedBy),
			wrap("createdBy", r.CreatedBy),
			wrap("updatedBy", r.UpdatedBy),
			wrap("deletedBy", r.DeletedBy),
		}

		for _, v := range r.Values {
			f := sch.Field(v.Name)
			if f == nil {
				return nil, fmt.Errorf("schema does not define field %s", v.Name)
			}

			builders = append(builders, wrap(f.Ident(), v.Value))
		}

		c, err := sch.BuildDocument(builders...)
		if err != nil {
			return nil, err
		}

		out = append(out, c)
	}

	return
}

func (crs *composeRecordStore) collectionToRecord(sch *schema.Collection, documents ...*schema.Document) (out types.RecordSet, err error) {
	out = make(types.RecordSet, len(documents))

	for i, doc := range documents {
		rec := &types.Record{
			Values: make(types.RecordValueSet, 0, 8),
		}

		err = sch.WalkValues(doc, func(k string, f schema.Field, vv []expr.TypedValue) error {
			switch k {
			case "recordID":
				// ...
			case "createdAt":
				// ...
			case "updatedAt":
				// ...
			case "deletedAt":
				// ...
			case "ownedBy":
				// ...
			case "createdBy":
				// ...
			case "updatedBy":
				// ...
			case "deletedBy":
				// ...
			}

			for i, v := range vv {
				recv := &types.RecordValue{
					Name:  k,
					Place: uint(i),
				}
				recv.Value, err = f.ToRV(v)
				if err != nil {
					return err
				}

				rec.Values = append(rec.Values, recv)
			}

			return nil
		})
		if err != nil {
			return
		}

		out[i] = rec
	}

	return
}
