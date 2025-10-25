package dao

import (
	"dst-management-platform-api/models"
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
	return &user, err
}

func (d *UserDAO) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := d.db.Where("email = ?", email).First(&user).Error
	return &user, err
}
