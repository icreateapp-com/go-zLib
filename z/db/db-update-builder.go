package db

import (
	"errors"
	"gorm.io/gorm"
)

type UpdateBuilder struct {
	Model interface{}
	TX    *gorm.DB
}

func (q UpdateBuilder) Update(search []ConditionGroup, values interface{}) (bool, error) {
	// db
	var db *gorm.DB
	if q.TX != nil {
		db = q.TX
	} else {
		db = DB.Model(q.Model)
	}

	// parse where clause
	db, err := QueryBuilder{q.Model}.ParseSearch(db, search, []string{})
	if err != nil {
		return false, err
	}

	if err := db.Updates(values).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (q UpdateBuilder) UpdateByID(id interface{}, values interface{}) (bool, error) {

	exists, _ := QueryBuilder{Model: q.Model}.ExistsById(id)
	if !exists {
		return false, errors.New("row not found")
	}

	return q.Update([]ConditionGroup{{
		Conditions: [][]interface{}{{"id", id}},
		Operator:   "AND",
	}}, values)
}
