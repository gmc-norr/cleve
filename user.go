package cleve

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

func (r UserRole) String() string {
	return string(r)
}

func ValidRole(role UserRole) bool {
	return slices.Contains([]UserRole{RoleAdmin, RoleUser}, UserRole(role))
}

type User struct {
	ID           uuid.UUID  `bson:"id"`
	DisplayName  string     `bson:"display_name"`
	Email        string     `bson:"email"`
	Roles        []UserRole `bson:"roles"`
	PasswordHash string     `bson:"password_hash"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
}
