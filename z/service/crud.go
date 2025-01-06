package service

import (
	. "github.com/icreateapp-com/go-zLib/z/db"
)

type CrudService struct {
	Model interface{}
}

func (s *CrudService) Get(query Query) (interface{}, error) {
	return QueryBuilder{Model: s.Model}.Page(query)
}

func (s *CrudService) Find(id interface{}, query Query) (interface{}, error) {
	return QueryBuilder{Model: s.Model}.FindById(id, query)
}

func (s *CrudService) Create(req interface{}) (interface{}, error) {
	return CreateBuilder{Model: s.Model}.Create(req)
}

func (s *CrudService) Update(id interface{}, req interface{}) (bool, error) {
	return UpdateBuilder{Model: s.Model}.UpdateByID(id, req)
}

func (s *CrudService) Delete(id interface{}) (bool, error) {
	return DeleteBuilder{Model: s.Model}.DeleteByID(id)
}
