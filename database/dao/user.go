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

func (d *UserDAO) ListUsers(q string, page, pageSize int) (*PaginatedResult[models.User], error) {
	var (
		condition string
		args      []any
	)
	if q != "" {
		searchUsername := "%" + q + "%"
		searchNickname := "%" + q + "%"
		condition = "username LIKE ? OR nickname LIKE ?"
		args = []any{searchUsername, searchNickname}
	}

	rooms, err := d.Query(page, pageSize, condition, args...)
	return rooms, err
}

func (d *UserDAO) UpdateUser(user *models.User) error {
	err := d.db.Save(user).Error
	return err
}

func (d *UserDAO) GetNonAdminUsers() (*[]models.User, error) {
	var users []models.User
	err := d.db.Where("role != 'admin'").Find(&users).Error

	return &users, err
}
