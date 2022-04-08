package crs

import (
	"context"

	"github.com/cortezaproject/corteza-server/compose/crs/capabilities"
	"github.com/cortezaproject/corteza-server/compose/crs/schema"
	"github.com/cortezaproject/corteza-server/compose/types"
)

func (crs *composeRecordStore) ComposeRecordCreate(ctx context.Context, mod *types.Module, records ...*types.Record) (err error) {
	// Determine required capabilities
	requiredCap := capabilities.CreateCapabilities(mod.Store.Capabilities)

	// Determine store
	var s Store
	if s, _, err = crs.getStore(ctx, mod.Store.ComposeRecordStoreID, requiredCap...); err != nil {
		return err
	}

	// Prepare data
	sch, err := crs.getModuleSchema(mod)
	if err != nil {
		return
	}
	cc, err := crs.recordToCollection(sch, records...)
	if err != nil {
		return
	}

	return s.CreateRecords(ctx, sch, cc...)
}

func (crs *composeRecordStore) ComposeRecordSearch(ctx context.Context, mod *types.Module, filter *types.RecordFilter) (records types.RecordSet, outFilter *types.RecordFilter, err error) {
	// Determine requiredCap we'll need
	requiredCap := capabilities.SearchCapabilities(mod.Store.Capabilities).Union(recFilterCapabilities(filter))

	// Connect to datasource
	var s Store
	var cc capabilities.Set
	s, cc, err = crs.getStore(ctx, mod.Store.ComposeRecordStoreID, requiredCap...)
	if err != nil {
		return
	}

	// Prepare data
	sch, err := crs.getModuleSchema(mod)
	if err != nil {
		return
	}
	loader, err := s.SearchRecords(ctx, sch, nil)
	if err != nil {
		return
	}

	limit := int(filter.Limit)
	if limit == 0 {
		limit = 10
	}

	auxCC := make([]*schema.Document, limit)

	var ok bool
	for loader.More() && len(records) < int(limit) {
		_, err = loader.Load(sch, auxCC)
		if err != nil {
			return
		}

		auxRecords, err := crs.collectionToRecord(sch, auxCC...)
		if err != nil {
			return nil, nil, err
		}

		for _, r := range auxRecords {
			r.ModuleID = mod.ID
			r.NamespaceID = mod.NamespaceID
		}

		if !cc.Can(capabilities.RBAC) && filter.Check != nil {
			for _, r := range auxRecords {
				if ok, err = filter.Check(r); err != nil {
					return nil, nil, err
				} else if !ok {
					continue
				}

				records = append(records, r)
			}
		} else if cc.Can(capabilities.RBAC) {
			for _, r := range auxRecords {
				if r == nil {
					break
				}
				records = append(records, r)
			}
		}
	}

	return
}

// ---

func recFilterCapabilities(f *types.RecordFilter) (out capabilities.Set) {
	if f == nil {
		return
	}
	if f.PageCursor != nil {
		out = append(out, capabilities.Paging)
	}

	if f.IncPageNavigation {
		out = append(out, capabilities.Paging)
	}

	if f.IncTotal {
		out = append(out, capabilities.Stats)
	}

	if f.Sort != nil {
		out = append(out, capabilities.Sorting)
	}

	return
}
