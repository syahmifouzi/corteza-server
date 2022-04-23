package rdbms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/filter"
	"github.com/cortezaproject/corteza-server/store"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

func MigrateComposeRecords(ctx context.Context, s *Store) (err error) {
	for _, tbl := range []string{"compose_record", "compose_record_value"} {
		if exists, err := s.SchemaAPI.TableExists(ctx, s.DB, tbl); err != nil {
			return fmt.Errorf("compose record migration failed: %w", err)
		} else if !exists {
			s.log(ctx).Debug("compose record migration skipped, " + tbl + " table does not exist")
			return nil
		}
	}

	type (
		// transitional record structure that will help us migrated
		// old (2-table) record storage into a new 1-table-with-json-encoded-values.
		tRecord struct {
			ID          uint64
			ModuleID    uint64
			NamespaceID uint64

			Values map[string][]any

			OwnedBy   uint64
			CreatedAt time.Time
			CreatedBy uint64
			UpdatedAt *time.Time
			UpdatedBy uint64
			DeletedAt *time.Time
			DeletedBy uint64
		}
	)

	var (
		chunk = 100

		mm types.ModuleSet
		ff types.ModuleFieldSet

		legacyRecTable = exp.ParseIdentifier("compose_record")
		legacyValTable = exp.ParseIdentifier("compose_record_value")
		migratedTable  = exp.ParseIdentifier("compose_records")

		countRecords = func(table exp.IdentifierExpression, ee ...exp.Expression) (c uint, _ error) {
			var (
				aux = struct {
					Count uint `db:"count"`
				}{}

				query = s.Dialect.
					From(table).
					Select(goqu.COUNT(goqu.Star()).As("count")).
					Where(ee...).
					Limit(1)
			)

			if err != nil {
				return
			}

			if err = s.QueryOne(ctx, query, &aux); err != nil {
				return
			}

			return aux.Count, nil
		}

		countLegacyRecords = func(moduleID uint64) (c uint, _ error) {
			return countRecords(legacyRecTable, exp.Ex{"module_id": moduleID})
		}

		countMigratedRecords = func(moduleID uint64) (c uint, _ error) {
			return countRecords(migratedTable, exp.Ex{"rel_module": moduleID})
		}

		loadLegacyRecords = func(s *Store, moduleID, lastRecordID uint64) ([]*tRecord, error) {
			var (
				query = s.Dialect.
					Select(
						"id",
						"module_id",
						"rel_namespace",
						"owned_by",
						"created_at",
						"created_by",
						"updated_at",
						"updated_by",
						"deleted_at",
						"deleted_by",
					).
					From(legacyRecTable).
					Where(
						exp.ParseIdentifier("module_id").Eq(moduleID),
						exp.ParseIdentifier("id").Gt(lastRecordID),
					).
					Order(exp.NewOrderedExpression(exp.ParseIdentifier("id"), exp.AscDir, exp.NoNullsSortType)).
					Limit(uint(chunk))

				rows, err = s.Query(ctx, query)
			)

			if err != nil {
				return nil, err
			}

			set := make([]*tRecord, 0, chunk)
			for rows.Next() {
				lr := new(tRecord)
				err = rows.Scan(
					&lr.ID, &lr.ModuleID, &lr.NamespaceID, &lr.OwnedBy,
					&lr.CreatedAt, &lr.CreatedBy,
					&lr.UpdatedAt, &lr.UpdatedBy,
					&lr.DeletedAt, &lr.DeletedBy,
				)

				if err != nil {
					return nil, err
				}

				lr.Values = make(map[string][]any)

				set = append(set, lr)
			}

			return set, nil
		}

		loadLegacyRecordValues = func(s *Store, mod *types.Module, rr []*tRecord) error {
			var (
				index     = make(map[uint64]int)
				recordIDs = make([]uint64, 0)
			)

			for i, r := range rr {
				index[r.ID] = i
				recordIDs = append(recordIDs, r.ID)
			}

			var (
				query = s.Dialect.
					Select("record_id", "name", "value").
					From(legacyValTable).
					Where(exp.ParseIdentifier("record_id").In(recordIDs))

				rows, err = s.Query(ctx, query)
			)

			if err != nil {
				return err
			}

			for rows.Next() {
				var (
					recordID    uint64
					name, value string
					v           any
				)

				if err = rows.Scan(&recordID, &name, &value); err != nil {
					return err
				}

				i, has := index[recordID]
				if !has {
					continue
				}

				f := mod.Fields.FindByName(name)
				if f == nil {
					continue
				}

				switch {
				case f.IsBoolean():
					v = cast.ToBool(v)
				default:
					v = value
				}

				rr[i].Values[name] = append(rr[i].Values[name], v)
			}

			return nil
		}
	)

	mm, _, err = store.SearchComposeModules(ctx, s, types.ModuleFilter{Deleted: filter.StateInclusive})
	if err != nil {
		return
	}

	ff, _, err = store.SearchComposeModuleFields(ctx, s, types.ModuleFieldFilter{Deleted: filter.StateInclusive})
	if err != nil {
		return
	}

	s.log(ctx).Info("migrating records from all modules", zap.Int("countModules", len(mm)))

	return mm.Walk(func(m *types.Module) (err error) {
		var (
			legacyCount, migratedCount uint

			log = s.log(ctx).With(
				zap.String("moduleHandle", m.Handle),
				zap.String("module", m.Name),
				zap.Uint64("moduleID", m.ID),
			)
		)

		if legacyCount, err = countLegacyRecords(m.ID); err != nil {
			return
		} else if legacyCount == 0 {
			log.Info("no legacy records found")

			return nil
		}

		if migratedCount, err = countMigratedRecords(m.ID); err != nil {
			return err
		} else if migratedCount > 0 {
			log.Info("record migrated", zap.Uint("count", migratedCount))

			return nil
		}

		m.Fields = ff.FilterByModule(m.ID)
		if len(m.Fields) == 0 {
			log.Info("no fields on module")
			return nil
		}

		log.Info("processing legacy records", zap.Uint("count", legacyCount))

		// start with from zero 0
		// we'll point to the last ID
		lastRecordID := uint64(0)
		return s.Tx(ctx, func(ctx context.Context, s store.Storer) (err error) {
			var (
				vJSON []byte
				set   []*tRecord
				db    = s.(any).(*Store).DB
			)
			_ = db

			for {
				log.Info("loading chunk of legacy records")
				if set, err = loadLegacyRecords(s.(any).(*Store), m.ID, lastRecordID); err != nil {
					return
				}

				if err = loadLegacyRecordValues(s.(any).(*Store), m, set); err != nil {
					return
				}

				for _, r := range set {
					if vJSON, err = json.Marshal(r.Values); err != nil {
						return
					}

					cmd := s.(any).(*Store).Dialect.
						Insert(migratedTable).
						Rows(goqu.Record{
							"id":            r.ID,
							"rel_module":    r.ModuleID,
							"rel_namespace": r.NamespaceID,
							"owned_by":      r.OwnedBy,
							"created_at":    r.CreatedAt,
							"created_by":    r.CreatedBy,
							"updated_at":    r.UpdatedAt,
							"updated_by":    r.UpdatedBy,
							"deleted_at":    r.DeletedAt,
							"deleted_by":    r.DeletedBy,
							"values":        vJSON,
						})

					if err = s.(any).(*Store).Exec(ctx, cmd); err != nil {
						return
					}
				}

				if len(set) < chunk {
					// did not fetch the full chunk, assuming all done.
					break
				}

				// move the needle for the next load/iteration
				lastRecordID = set[len(set)-1].ID
			}

			return nil
		})
	})
}
