package db_middlewares

import "gorm.io/gorm"

type Middleware func(db *gorm.DB) error
