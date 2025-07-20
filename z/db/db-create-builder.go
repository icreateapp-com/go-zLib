package db

import "gorm.io/gorm"

type CreateBuilder[T IModel] struct {
	TX *gorm.DB
}

func (q CreateBuilder[T]) Create(values T) (T, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		db = DB.Model(&zero)
	}

	if err := db.Create(&values).Error; err != nil {
		return zero, err
	}

	return values, nil
}
