package crud_controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/controller/base_controller"
	"github.com/icreateapp-com/go-zLib/z/db"
	"github.com/icreateapp-com/go-zLib/z/service/crud_service"
)

// ICrudController CRUD控制器接口
type ICrudController[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any] interface {
	Get(c *gin.Context)
	Find(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

// ServiceFactory 服务工厂函数类型
type ServiceFactory[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any] func(ctx context.Context) crud_service.ICrudService[TModel, TCreateRequest, TUpdateRequest, TResponse]

// CrudController 通用CRUD控制器
type CrudController[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any] struct {
	base_controller.BaseController
	serviceFactory ServiceFactory[TModel, TCreateRequest, TUpdateRequest, TResponse]
	BeforeCreate   func(c *gin.Context, req *TCreateRequest) error                      // 创建前的钩子函数
	BeforeUpdate   func(c *gin.Context, req *TUpdateRequest) error                      // 更新前的钩子函数
	BeforeDelete   func(c *gin.Context, id string) error                                // 删除前的钩子函数
	AfterCreated   func(c *gin.Context, req *TCreateRequest, res *TResponse)            // 创建后的钩子函数
	AfterUpdated   func(c *gin.Context, id string, req *TUpdateRequest, res *TResponse) // 更新后的钩子函数
	AfterDeleted   func(c *gin.Context, id string)                                      // 删除后的钩子函数
}

// Get 获取数据列表
func (ctrl *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]) Get(c *gin.Context) {
	query := ctrl.GetQuery(c)
	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Page(query); err != nil {
		z.Failure(c, err)
	} else {
		z.Success(c, res)
	}
}

// Find 获取单个数据
func (ctrl *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]) Find(c *gin.Context) {
	id := c.Param("id")
	query := ctrl.GetQuery(c)
	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Find(id, query); err != nil {
		z.Failure(c, err)
	} else {
		z.Success(c, res)
	}
}

// Create 创建数据
func (ctrl *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]) Create(c *gin.Context) {
	var req TCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		z.Failure(c, z.Validator.T(err, req))
		return
	}

	if ctrl.BeforeCreate != nil {
		if err := ctrl.BeforeCreate(c, &req); err != nil {
			z.Failure(c, err)
			return
		}
	}

	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Create(&req); err != nil {
		z.Failure(c, err)
	} else {
		if ctrl.AfterCreated != nil {
			ctrl.AfterCreated(c, &req, res)
		}
		z.Success(c, res)
	}
}

// Update 更新数据
func (ctrl *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]) Update(c *gin.Context) {
	id := c.Param("id")
	var req TUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		z.Failure(c, z.Validator.T(err, req))
		return
	}

	if ctrl.BeforeUpdate != nil {
		if err := ctrl.BeforeUpdate(c, &req); err != nil {
			z.Failure(c, err)
			return
		}
	}

	service := ctrl.serviceFactory(c.Request.Context())
	if res, err := service.Update(id, &req); err != nil {
		z.Failure(c, err)
	} else {
		if ctrl.AfterUpdated != nil {
			ctrl.AfterUpdated(c, id, &req, res)
		}
		z.Success(c, res)
	}
}

// Delete 删除数据
func (ctrl *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]) Delete(c *gin.Context) {
	id := c.Param("id")

	if ctrl.BeforeDelete != nil {
		if err := ctrl.BeforeDelete(c, id); err != nil {
			z.Failure(c, err)
			return
		}
	}

	service := ctrl.serviceFactory(c.Request.Context())
	if _, err := service.DeleteByID(id); err != nil {
		z.Failure(c, err)
	} else {
		if ctrl.AfterDeleted != nil {
			ctrl.AfterDeleted(c, id)
		}
		z.Success(c)
	}
}

// New 创建新的CRUD控制器
func New[TModel db.IModel, TCreateRequest any, TUpdateRequest any, TResponse any](factory ServiceFactory[TModel, TCreateRequest, TUpdateRequest, TResponse]) *CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse] {
	return &CrudController[TModel, TCreateRequest, TUpdateRequest, TResponse]{
		serviceFactory: factory,
	}
}
