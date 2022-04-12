package postgres

import (
	"context"
	"database/sql"
	"net/url"
	"strings"

	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/cortezaproject/corteza-server/store"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/ddl"
	"github.com/cortezaproject/corteza-server/store/adapters/rdbms/instrumentation"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ngrok/sqlmw"
)

func init() {
	store.Register(Connect, "postgres", "postgres+debug")
	sql.Register("postgres+debug", sqlmw.Driver(new(pq.Driver), instrumentation.Debug()))
}

func Connect(ctx context.Context, dsn string) (_ store.Storer, err error) {
	var (
		db  *sqlx.DB
		cfg *rdbms.ConnConfig
	)

	if cfg, err = NewConfig(dsn); err != nil {
		return
	}

	if db, err = rdbms.Connect(ctx, logger.Default(), cfg); err != nil {
		return
	}

	s := &rdbms.Store{
		DB: db,

		Dialect:      goqu.Dialect("postgres"),
		ErrorHandler: errorHandler,

		// @todo this should probably by pgsql dialect
		SchemaAPI: &SchemaModifier{
			GenericSchemaModifier: &ddl.GenericSchemaModifier{
				Dialect: ddl.NewCommonDialect(),
			},
		},
	}

	s.SetDefaults()

	return s, nil
}

// ProcDataSourceName validates given DSN and ensures
// params are present and correct
func NewConfig(dsn string) (c *rdbms.ConnConfig, err error) {
	const (
		validScheme = "postgres"
	)
	var (
		scheme string
		u      *url.URL
	)

	if u, err = url.Parse(dsn); err != nil {
		return nil, err
	}

	if strings.HasPrefix(dsn, "postgres") {
		scheme = u.Scheme
		u.Scheme = validScheme
	}

	c = &rdbms.ConnConfig{
		DriverName:     scheme,
		DataSourceName: u.String(),
		DBName:         strings.Trim(u.Path, "/"),
	}

	c.SetDefaults()

	return c, nil
}

func errorHandler(err error) error {
	if err != nil {
		if implErr, ok := err.(*pq.Error); ok {
			switch implErr.Code.Name() {
			case "unique_violation":
				return store.ErrNotUnique.Wrap(implErr)
			}
		}
	}

	return err
}
