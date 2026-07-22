package orm

import (
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type PostgresqlConfig struct {
	Dsn          string
	TimeZone     string
	MaxIdleConns int
	MaxOpenConns int
}

func NewPostgresql(conf *PostgresqlConfig, logwriter logger.Writer) (*gorm.DB, error) {
	pgxConfig, err := pgx.ParseConfig(conf.Dsn)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL connection configuration: %w", err)
	}
	if conf.TimeZone != "" {
		pgxConfig.RuntimeParams["timezone"] = conf.TimeZone
	}
	sqlDB := stdlib.OpenDB(*pgxConfig)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold: time.Second, // Slow SQL threshold
				LogLevel:      logger.Warn, // Log level
				//IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries: true, // Don't include params in the SQL log
				Colorful:             true,
			},
		),
	})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	gormSQLDB, err := db.DB()
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	gormSQLDB.SetMaxIdleConns(conf.MaxIdleConns)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	gormSQLDB.SetMaxOpenConns(conf.MaxOpenConns)

	return db, nil
}
