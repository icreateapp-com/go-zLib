package db

import (
	"fmt"

	"github.com/icreateapp-com/go-zLib/z"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDriver struct {
	db *gorm.DB
}

func NewMySQLDriver() gorm.Dialector {
	user, _ := z.Config.String("config.db.username")
	pass, _ := z.Config.String("config.db.password")
	host, _ := z.Config.String("config.db.host")
	port, _ := z.Config.String("config.db.port")
	name, _ := z.Config.String("config.db.dbname")
	char, _ := z.Config.String("config.db.charset")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local", user, pass, host, port, name, char)
	return mysql.Open(dsn)
}
