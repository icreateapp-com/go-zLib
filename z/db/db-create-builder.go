package db

import "gorm.io/gorm"

type CreateBuilder struct {
	Model interface{}
	TX    *gorm.DB
}

func (q CreateBuilder) Create(values interface{}) (interface{}, error) {
	// db
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX
	} else {
		db = DB.Model(q.Model)
	}

	if err := db.Create(values).Error; err != nil {
		return nil, err
	}

	return values, nil
}
