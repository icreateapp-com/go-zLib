package db

type CreateBuilder struct {
	Model interface{}
}

func (q CreateBuilder) Create(values interface{}) (interface{}, error) {
	// db
	db := DB.Model(q.Model)

	if err := db.Create(values).Error; err != nil {
		return nil, err
	}

	return values, nil
}
