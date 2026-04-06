package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid role")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRotation      = errors.New("token rotation failed")
	ErrPasswordChange     = errors.New("password change failed")
	ErrUserDelete         = errors.New("user deletion failed")
)
