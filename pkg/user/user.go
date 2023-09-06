package user

import (
	"time"
)

type User struct {
	Id        string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (u *User) Update(options *UpdateOptions, fieldMask *UpdateFieldMask) {
	if fieldMask.Email {
		u.Email = options.Email
	}

	if fieldMask.Password {
		u.Password = options.Password
	}
}
