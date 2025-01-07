package db

type DeleteBuilder struct {
	Model interface{}
}

func (q DeleteBuilder) Delete(search []ConditionGroup) (bool, error) {
	// db
	db := DB.Model(q.Model)

	// parse where clause
	db, err := QueryBuilder{q.Model}.ParseSearch(db, search, []string{})
	if err != nil {
		return false, err
	}

	if err := db.Delete(q.Model).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (q DeleteBuilder) DeleteByID(id interface{}) (bool, error) {
	return q.Delete([]ConditionGroup{{
		Conditions: [][]interface{}{{"id", id}},
		Operator:   "AND",
	}})
}
