package dao

import (
	"dst-management-platform-api/database/models"
	"errors"
	"gorm.io/gorm"
)

type UserDAO struct {
	BaseDAO[models.User]
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		BaseDAO: *NewBaseDAO[models.User](db),
	}
}

func (d *UserDAO) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := d.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &user, nil
	}
	return &user, err
}

func (d *UserDAO) ListUsers(username string, page, pageSize int) (*PaginatedResult[models.User], error) {
	var (
		condition string
		args      []interface{}
	)
	if username != "" {
		searchPattern := "%" + username + "%"
		condition = "username LIKE ?"
		args = []interface{}{searchPattern}
	}

	rooms, err := d.Query(page, pageSize, condition, args...)
	return rooms, err
}
