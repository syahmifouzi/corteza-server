package sqlserver

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
	_ "github.com/denisenkom/go-mssqldb"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlserver"
	"github.com/jmoiron/sqlx"
	"github.com/ngrok/sqlmw"
)

func init() {
	store.Register(Connect, "sqlserver", "sqlserver+debug")
	sql.Register("sqlserver+debug", sqlmw.Driver(new(mssql.Driver), instrumentation.Debug()))
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

		Dialect:      goqu.Dialect("sqlserver"),
		ErrorHandler: errorHandler,

		// @todo this should probably by sqlserver dialect
		SchemaAPI: &ddl.GenericSchemaModifier{Dialect: ddl.NewCommonDialect()},
	}

	s.SetDefaults()

	return s, nil
}

// NewConfig validates given DSN and ensures
// params are present and correct
func NewConfig(dsn string) (_ *rdbms.ConnConfig, err error) {
	const (
		validScheme = "sqlserver"
	)
	var (
		scheme string
		u      *url.URL
		c      = &rdbms.ConnConfig{DriverName: scheme}
	)

	if u, err = url.Parse(dsn); err != nil {
		return nil, err
	}

	if strings.HasPrefix(dsn, "sqlserver") {
		scheme = u.Scheme
		u.Scheme = validScheme
	}

	c.DataSourceName = u.String()
	c.DBName = strings.Trim(u.Path, "/")

	c.SetDefaults()

	return c, nil
}

func errorHandler(err error) error {
	if err != nil {
		panic(err)
		//if implErr, ok := err.(*pq.Error); ok {
		//	switch implErr.Code.Name() {
		//	case "unique_violation":
		//		return store.ErrNotUnique.Wrap(implErr)
		//	}
		//}
	}

	return err
}
