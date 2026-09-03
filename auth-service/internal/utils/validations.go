package utils

// File dedicated to User related validations

import (
	"errors"
	"go-task-wallet-service/auth-service/internal/domain"
	"regexp"
	"strings"
	"unicode"
)

// Errors based on the failed validaton
var (
	ErrNameRequired     = errors.New("name is required")
	ErrUsernameRequired = errors.New("username is required")
	ErrUsernameInvalid  = errors.New("username must be 3-20 characters, alphanumeric and underscores only")
	ErrEmailRequired    = errors.New("email is required")
	ErrEmailInvalid     = errors.New("email format is invalid")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordWeak     = errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// Validate runs basic sanity checks on a User before registration.
func ValidateUser(user *domain.User) error {
	user.Name = strings.TrimSpace(user.Name)
	user.Username = strings.TrimSpace(user.Username)
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))

	if user.Name == "" {
		return ErrNameRequired
	}

	if user.Username == "" {
		return ErrUsernameRequired
	}
	if !usernameRegex.MatchString(user.Username) {
		return ErrUsernameInvalid
	}

	if user.Email == "" {
		return ErrEmailRequired
	}
	if !emailRegex.MatchString(user.Email) {
		return ErrEmailInvalid
	}

	if err := validatePassword(user.Password); err != nil {
		return err
	}

	return nil
}

// Validating user password based on given categories
// Those categories include upper/lowercase presence. number presence, special character presence
func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrPasswordWeak
	}

	return nil
}
