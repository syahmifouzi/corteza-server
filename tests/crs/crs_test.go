package qwe

import (
	"context"
	"testing"

	"github.com/cortezaproject/corteza-server/compose/crs"
	"github.com/cortezaproject/corteza-server/compose/crs/capabilities"
	"github.com/cortezaproject/corteza-server/compose/crs/testko"
	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/stretchr/testify/require"
)

func TestCRS(t *testing.T) {
	// Meta bits
	ctx := context.Background()

	mod := &types.Module{
		ID:     21000,
		Handle: "a_module",
		Name:   "A Module",
		Store: types.CRSDef{
			ComposeRecordStoreID: 0,
		},
		Fields: types.ModuleFieldSet{{
			Name: "name",
			Kind: "String",
		}, {
			Name: "email",
			Kind: "Email",
		}},
	}

	rec := &types.Record{
		ID:       31000,
		ModuleID: mod.ID,
		Values: types.RecordValueSet{{
			Name:  "name",
			Value: "My Name",
		}, {
			Name:  "email",
			Value: "qwerty@test.tld",
		}},
	}

	crsDef := types.ComposeRecordStore{
		ID:        0,
		Handle:    "qwerty",
		DSN:       "testko://corteza.something",
		Supported: capabilities.FullCapabilities(),
	}

	//
	//
	//

	// Init CRS with available drivers and default store

	s, err := crs.ComposeRecordStore(
		ctx,
		crsDef,
		testko.Driver(),
	)
	require.NoError(t, err)

	// @todo schema related stuff

	//
	//
	//

	// To create records

	err = s.ComposeRecordCreate(ctx, mod, rec)
	require.NoError(t, err)

	// To search records

	_, _, err = s.ComposeRecordSearch(ctx, mod, nil)
	require.NoError(t, err)

	require.FailNow(t, "wip")
}
