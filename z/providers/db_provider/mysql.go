package db_provider

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDriver struct {
	db *gorm.DB
}

type MySQLConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	DBName   string
	Charset  string
}

func NewMySQLDialector(cfg MySQLConfig) gorm.Dialector {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	charset := strings.TrimSpace(cfg.Charset)
	if charset == "" {
		charset = "utf8mb4"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, host, port, cfg.DBName, charset)
	return mysql.Open(dsn)
}
