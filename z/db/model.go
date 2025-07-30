package db

import (
	googleUuid "github.com/google/uuid"
	"gorm.io/gorm"
)

// IModel 模型接口
// 所有模型必须继承
type IModel interface {
	TableName() string
}

// BaseModel 模型基类
type BaseModel struct {
}

// Timestamp 模型时间戳
type Timestamp struct {
	CreatedAt WrapTime `gorm:"type:datetime;autoCreateTime;<-:create" json:"created_at"`
	UpdatedAt WrapTime `gorm:"type:datetime;autoCreateTime;autoUpdateTime" json:"updated_at"`
}

// SoftDelete 软删除模型
type SoftDelete struct {
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type AutoIncrement struct {
	ID int64 `json:"id" gorm:"unique;primaryKey;autoIncrement;->;<-:false"`
}

type Uuid struct {
	ID string `gorm:"unique;primaryKey" json:"id" form:"id"`
}

func (m *Uuid) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = googleUuid.New().String()
	return
}
