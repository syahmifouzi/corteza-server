package crs

import (
	"context"

	"github.com/cortezaproject/corteza-server/compose/crs/schema"
	"github.com/cortezaproject/corteza-server/compose/types"
)

func (crs *composeRecordStore) AssertStore(ctx context.Context, cs cStore) (err error) {
	var (
		namespaces types.NamespaceSet
		modules    types.ModuleSet
		fields     types.ModuleFieldSet
	)

	namespaces, _, err = cs.SearchComposeNamespaces(ctx, types.NamespaceFilter{})
	if err != nil {
		return
	}

	for _, ns := range namespaces {
		modules, _, err = cs.SearchComposeModules(ctx, types.ModuleFilter{
			NamespaceID: ns.ID,
		})
		if err != nil {
			return
		}

		for _, mod := range modules {
			fields, _, err = cs.SearchComposeModuleFields(ctx, types.ModuleFieldFilter{
				ModuleID: []uint64{mod.ID},
			})
			if err != nil {
				return
			}

			mod.Fields = append(mod.Fields, fields...)
		}
	}

	if err = crs.assertModule(ctx, modules...); err != nil {
		return
	}
	return nil
}

func (crs *composeRecordStore) AssertModule(ctx context.Context, mod ...*types.Module) (err error) {
	return crs.assertModule(ctx, mod...)
}

func (crs *composeRecordStore) assertModule(ctx context.Context, modules ...*types.Module) (err error) {
	byStore := crs.modulesByStore(modules)
	var s Store

	for storeID, modules := range byStore {
		s, _, err = crs.getStore(ctx, storeID)
		if err != nil {
			return err
		}

		cc := make([]*schema.Collection, len(modules))
		for i, m := range modules {
			cc[i], err = crs.getModuleSchema(m)
			if err != nil {
				return
			}
		}

		err = s.AssertCollection(ctx, cc[0], cc[1:]...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (crs *composeRecordStore) modulesByStore(modules types.ModuleSet) (out map[uint64]types.ModuleSet) {
	out = make(map[uint64]types.ModuleSet)
	for _, mod := range modules {
		out[mod.Store.ComposeRecordStoreID] = append(out[mod.Store.ComposeRecordStoreID], mod)
	}

	return
}
