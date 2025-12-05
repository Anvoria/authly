package user

import "github.com/Anvoria/authly/internal/database"

type User struct {
	database.BaseModel
	Username  string `gorm:"column:username;unique;not null"`
	FirstName string `gorm:"column:first_name;not null"`
	LastName  string `gorm:"column:last_name;not null"`
	Email     string `gorm:"column:email;unique"`
	Password  string `gorm:"column:password;not null"`
	IsActive  bool   `gorm:"column:is_active;default:true"`
}

func (User) TableName() string {
	return "users"
}
