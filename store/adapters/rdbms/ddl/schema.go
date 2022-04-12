package ddl

import (
	"context"
	"database/sql"
	"fmt"
)

type (
	SqlExecer interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
		//SelectContext(context.Context, interface{}, string, ...interface{}) error
		GetContext(context.Context, interface{}, string, ...interface{}) error
		//QueryRowContext(context.Context, string, ...interface{}) *sql.Row
		//QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	}

	sqlGenerator interface {
		GenerateSQL(string, any) string
	}

	// GenericSchemaModifier provides API to the underlying SQL store to allow
	// creation, dropping and alteration of objects (tables, columns, indexes, ...)
	GenericSchemaModifier struct {
		// generates schemaAPI altering SQL
		Dialect sqlGenerator
	}

	SchemaModifier interface {
		Create(context.Context, SqlExecer, any) error
		TableExists(context.Context, SqlExecer, string) (bool, error)
	}
)

func (s *GenericSchemaModifier) Create(ctx context.Context, db SqlExecer, item any) (err error) {
	var ddlTemplate string

	switch item.(type) {
	case *Table:
		ddlTemplate = "create-table"
	default:
		return fmt.Errorf("no support for creating %T", item)
	}

	_, err = db.ExecContext(ctx, s.Dialect.GenerateSQL(ddlTemplate, item))
	return
}

//func (s *schemaAPI) RenameTable(ctx context.Context, old, new string) error {
//	if exists, err := s.TableExists(ctx, old); err != nil {
//		return err
//	} else if !exists {
//		s.log.Debug(fmt.Sprintf("skipping %s table rename, old table does not exist", old))
//		return nil
//	}
//
//	if exists, err := s.TableExists(ctx, new); err != nil {
//		return err
//	} else if exists {
//		s.log.Debug(fmt.Sprintf("skipping %s table rename, new table already exist", new))
//		return nil
//	}
//
//	// @todo move to sql-generator template
//	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", old, new)); err != nil {
//		return err
//	}
//
//	s.log.Debug(fmt.Sprintf("table %s renamed to %s", old, new))
//
//	return nil
//}

func (s *GenericSchemaModifier) TableExists(ctx context.Context, db SqlExecer, table string) (bool, error) {
	var tmp interface{}
	// @todo move to sql-generator template
	if err := db.GetContext(ctx, &tmp, fmt.Sprintf(`SHOW TABLES LIKE '%s'`, table)); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("could not check if table exists: %w", err)
	}

	return true, nil
}
