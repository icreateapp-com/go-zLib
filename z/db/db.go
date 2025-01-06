package db

import (
	"database/sql"
	"github.com/icreateapp-com/go-zLib/z"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strings"
	"time"
)

type db struct {
	*gorm.DB
}

var DB db

// New 初始化
func (db *db) New() *db {

	if db.DB != nil {
		return db
	}

	driver, err := z.Config.String("config.db.driver")
	driver = strings.ToLower(driver)
	if err != nil {
		z.Error.Fatal("unconfigured db type in config")
	}

	var dbDriver gorm.Dialector

	switch driver {
	case "mysql":
		dbDriver = NewMySQLDriver()
		break
	default:
		z.Error.Fatal("unknown db type in config")
	}

	debugLevel := logger.Error
	if debug, _ := z.Config.Bool("config.debug"); debug {
		debugLevel = logger.Info
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second * 5,
			LogLevel:                  debugLevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  true,
		},
	)

	db.DB, err = gorm.Open(dbDriver, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		z.Error.Fatal("db connect error: ", err.Error())
	}

	return db
}

// Transaction 事务
func (db *db) Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return db.DB.Transaction(fc, opts...)
}

// F 字段转义
func (db *db) F(field string) string {
	if db.Dialector.Name() == "mysql" {
		return "`" + field + "`"
	}
	return field
}
