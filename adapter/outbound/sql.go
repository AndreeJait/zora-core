package outbound

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/config"
	gormw "github.com/AndreeJait/go-utility/v2/sql/gormw"
	sqlxw "github.com/AndreeJait/go-utility/v2/sql/sqlxw"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// DB holds either a GORM or SQLX database connection, selected by config.
type DB struct {
	GormDB *gorm.DB
	SQLXDB *sqlx.DB
	Driver string // "gorm" or "sqlx"
}

// ConnectSQL establishes a database connection using the configured driver (gorm or sqlx).
// Returns a DB struct and a cleanup function compatible with gracefulw.Register().
func ConnectSQL(ctx context.Context, cfg *config.AppConfig) (*DB, func(ctx context.Context) error, error) {
	switch cfg.DB.Driver {
	case "gorm":
		db, err := gormw.Connect(ctx, &gormw.Config{
			Driver:          gormw.DriverType(cfg.DB.Dialect),
			DSN:             cfg.DB.DSN,
			MaxOpenConns:    cfg.DB.MaxOpenConns,
			MaxIdleConns:    cfg.DB.MaxIdleConns,
			ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
			DebugMode:       cfg.DB.DebugMode,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect gorm: %w", err)
		}
		return &DB{GormDB: db, Driver: "gorm"}, gormw.Disconnect(db), nil

	case "sqlx":
		db, err := sqlxw.Connect(ctx, &sqlxw.Config{
			Driver:          sqlxw.DriverType(dialectToSQLX(cfg.DB.Dialect)),
			DSN:             cfg.DB.DSN,
			MaxOpenConns:    cfg.DB.MaxOpenConns,
			MaxIdleConns:    cfg.DB.MaxIdleConns,
			ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
			DebugMode:       cfg.DB.DebugMode,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect sqlx: %w", err)
		}
		return &DB{SQLXDB: db, Driver: "sqlx"}, sqlxw.Disconnect(db), nil

	default:
		return nil, nil, fmt.Errorf("unknown db driver: %s (must be gorm or sqlx)", cfg.DB.Driver)
	}
}

// dialectToSQLX converts config dialect to sqlx driver name.
// sqlx uses "postgres" same as gorm, but uses "sqlite3" (with the 3).
func dialectToSQLX(dialect string) string {
	if dialect == "sqlite" {
		return "sqlite3"
	}
	return dialect
}