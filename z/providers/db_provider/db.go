package db_provider

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/icreateapp-com/go-zLib/z/providers/config_provider"
	"github.com/icreateapp-com/go-zLib/z/providers/db_provider/db_middlewares"
	"github.com/icreateapp-com/go-zLib/z/providers/logger_provider"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	log *logger_provider.Logger
}

type MiddlewaresIn struct {
	fx.In
	Registry *db_middlewares.Registry `optional:"true"`
}

// NewDBProvider 创建数据库连接（fx Provider）
func NewDBProvider(lc fx.Lifecycle, in MiddlewaresIn, cfg *config_provider.Config, log *logger_provider.Logger) (*DB, error) {
	driver := strings.ToLower(cfg.GetString("db.driver", "mysql"))

	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = NewMySQLDialector(MySQLConfig{
			Username: cfg.GetString("db.mysql.username"),
			Password: cfg.GetString("db.mysql.password"),
			Host:     cfg.GetString("db.mysql.host", "127.0.0.1"),
			Port:     cfg.GetInt("db.mysql.port", 3306),
			DBName:   cfg.GetString("db.mysql.dbname"),
			Charset:  cfg.GetString("db.mysql.charset", "utf8mb4"),
		})
	default:
		return nil, fmt.Errorf("unknown db type: %s", driver)
	}

	debugLevel := logger.Error
	if cfg.GetBool("app.debug", true) {
		debugLevel = logger.Info
	}

	std := zap.NewStdLog(log.Base())
	gormLogger := NewFilteredGormLogger(logger.New(
		std,
		logger.Config{
			SlowThreshold:             5 * time.Second,
			LogLevel:                  debugLevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  true,
		},
	))

	gdb, err := gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Errorw("db connect error", "error", err)
		return nil, err
	}

	db := &DB{DB: gdb, log: log}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			middlewares := cfg.GetStringSlice("db.middlewares", nil)
			if len(middlewares) > 0 {
				if in.Registry == nil {
					return fmt.Errorf("db.middlewares configured but registry is nil")
				}
				if err := in.Registry.Apply(gdb, middlewares); err != nil {
					return err
				}
			}

			sqlDB, err := gdb.DB()
			if err != nil {
				return err
			}
			applyDBPoolConfig(sqlDB, cfg)
			if err := sqlDB.PingContext(ctx); err != nil {
				return err
			}
			log.Infow(
				"provider[db] middlewares",
				"driver",
				driver,
				"max_open_conns",
				sqlDB.Stats().MaxOpenConnections,
				"max_idle_conns",
				cfg.GetInt("db.mysql.max_idle_conns", 10),
				"conn_max_lifetime",
				cfg.GetDuration("db.mysql.conn_max_lifetime", 5*time.Minute).String(),
				"conn_max_idle_time",
				cfg.GetDuration("db.mysql.conn_max_idle_time", 2*time.Minute).String(),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			sqlDB, err := gdb.DB()
			if err != nil {
				return nil
			}
			return sqlDB.Close()
		},
	})

	return db, nil
}

// applyDBPoolConfig 配置 GORM 底层 database/sql 连接池。
//
// 这是 Go / GORM 官方推荐方式：通过 gorm.DB() 取得 *sql.DB 后设置连接池参数，
// 避免长生命周期服务反复复用已被 MySQL / 代理层断开的空闲连接。
func applyDBPoolConfig(sqlDB *sql.DB, cfg *config_provider.Config) {
	if sqlDB == nil || cfg == nil {
		return
	}

	maxOpenConns := cfg.GetInt("db.mysql.max_open_conns", 50)
	if maxOpenConns <= 0 {
		maxOpenConns = 50
	}

	maxIdleConns := cfg.GetInt("db.mysql.max_idle_conns", 10)
	if maxIdleConns < 0 {
		maxIdleConns = 10
	}
	if maxIdleConns > maxOpenConns {
		maxIdleConns = maxOpenConns
	}

	connMaxLifetime := cfg.GetDuration("db.mysql.conn_max_lifetime", 5*time.Minute)
	if connMaxLifetime <= 0 {
		connMaxLifetime = 5 * time.Minute
	}

	connMaxIdleTime := cfg.GetDuration("db.mysql.conn_max_idle_time", 2*time.Minute)
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 2 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
}

// DBProviderModule 数据库模块
var DBProviderModule = fx.Options(
	db_middlewares.RegistryModule,
	db_middlewares.OtelGormModule,
	db_middlewares.CachesModule,
	fx.Provide(NewDBProvider),
)

// Transaction 事务装饰器 - 自动管理事务生命周期
func (db *DB) Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return db.DB.Transaction(fc, opts...)
}

// F 字段转义
func (db *DB) F(field string) string {
	if db.Dialector.Name() == "mysql" {
		return "`" + field + "`"
	}
	return field
}
