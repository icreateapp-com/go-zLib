package crud_service

import (
	"github.com/icreateapp-com/go-zLib/z/db"
)

// SimpleCrudService is a simplified version of CrudService that uses the same type for model, create request, update request, and response.
type SimpleCrudService[T db.IModel] struct {
	CrudService[T, T, T, T]
}