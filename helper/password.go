package helper

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// dummyPasswordHash is compared against when a login identifier does not exist,
// so the response time does not reveal whether the account exists.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("secure-patrol-dummy-password"), bcrypt.DefaultCost)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func CheckDummyPassword(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
}

var ErrPasswordWeak = errors.New("password must be 8-72 characters and contain at least one letter and one digit")

// ValidatePasswordStrength requires 8-72 characters with at least one letter and one digit.
// 72 is the maximum input length bcrypt accepts.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrPasswordWeak
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return ErrPasswordWeak
	}

	return nil
}
