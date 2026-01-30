package dao

import (
	"gorm.io/gorm"
)

type BaseDAO[T any] struct {
	db *gorm.DB
}

type PaginatedResult[T any] struct {
	Data       []T   `json:"rows"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalCount int64 `json:"total"`
}

func NewBaseDAO[T any](db *gorm.DB) *BaseDAO[T] {
	return &BaseDAO[T]{db: db}
}

func (d *BaseDAO[T]) Create(model *T) error {
	return d.db.Create(model).Error
}

func (d *BaseDAO[T]) Update(model *T) error {
	return d.db.Save(model).Error
}

func (d *BaseDAO[T]) Delete(model *T) error {
	return d.db.Delete(model).Error
}

func (d *BaseDAO[T]) Query(page, pageSize int, condition any, args ...any) (*PaginatedResult[T], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var models []T
	var total int64

	query := d.db.Model(new(T))
	if condition != nil {
		query = query.Where(condition, args...)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Find(&models).Error

	return &PaginatedResult[T]{
		Data:       models,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
	}, err
}

func (d *BaseDAO[T]) Count(condition any, args ...any) (int64, error) {
	var count int64
	err := d.db.Model(new(T)).Where(condition, args...).Count(&count).Error

	return count, err
}
