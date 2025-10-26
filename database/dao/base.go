package dao

import (
	"gorm.io/gorm"
)

type BaseDAO[T any] struct {
	db *gorm.DB
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

func (d *BaseDAO[T]) FindAll() ([]T, error) {
	var models []T
	err := d.db.Find(&models).Error
	return models, err
}

func (d *BaseDAO[T]) Query(condition interface{}, args ...interface{}) ([]T, error) {
	var models []T
	err := d.db.Where(condition, args...).Find(&models).Error
	return models, err
}

func (d *BaseDAO[T]) Count(condition interface{}, args ...interface{}) (int64, error) {
	var count int64
	err := d.db.Model(new(T)).Where(condition, args...).Count(&count).Error
	return count, err
}
