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

	users, err := d.Query(page, pageSize, condition, args...)

	return users, err
}

func (d *UserDAO) UpdateUser(user *models.User) error {
	err := d.db.Save(user).Error

	return err
}

func (d *UserDAO) UpdatePassword(username, password, passwordVersion string) error {
	result := d.db.Model(&models.User{}).
		Where("username = ?", username).
		Updates(map[string]any{
			"password":         password,
			"password_version": passwordVersion,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (d *UserDAO) GetNonAdminUsers() (*[]models.User, error) {
	var users []models.User
	err := d.db.Where("role != 'admin'").Find(&users).Error

	return &users, err
}
