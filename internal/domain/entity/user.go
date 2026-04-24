package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email         string    `gorm:"uniqueIndex;not null"`
	PasswordHash  *string
	GoogleID      *string `gorm:"uniqueIndex"`
	IsVerified    bool
	TOTPSecret    *string
	TOTPEnabled   bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Roles         []Role `gorm:"many2many:user_roles"`
}

func (u *User) TableName() string {
	return "users"
}

func NewUser(email string) *User {
	return &User{
		ID:          uuid.New(),
		Email:       email,
		IsVerified:  false,
		TOTPEnabled: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
