package db

import "gorm.io/gorm"

type CreateBuilder[T IModel] struct {
	TX *gorm.DB
}

func (q CreateBuilder[T]) Create(values T, customFunc ...func(*gorm.DB) *gorm.DB) (T, error) {
	var zero T
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX.Model(&zero)
	} else {
		db = DB.Model(&zero)
	}

	// 应用自定义函数（如 Select、Omit、OnConflict 等）
	for _, fn := range customFunc {
		if fn != nil {
			db = fn(db)
		}
	}

	// 创建一个副本用于数据库操作，确保原始数据不被修改
	result := values
	if err := db.Create(&result).Error; err != nil {
		return zero, err
	}

	// 返回包含自动生成字段（如 ID）的结果
	return result, nil
}
