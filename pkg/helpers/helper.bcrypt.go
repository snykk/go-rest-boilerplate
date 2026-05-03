package helpers

import (
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost returns the configured cost, falling back to
// bcrypt.DefaultCost when config hasn't been initialized (tests, tools
// that import helpers without loading .env). Clamped to bcrypt's own
// valid range so a bad config can't make this panic.
func bcryptCost() int {
	c := config.AppConfig.BcryptCost
	if c < bcrypt.MinCost || c > bcrypt.MaxCost {
		return bcrypt.DefaultCost
	}
	return c
}

func GenerateHash(passwd string) (string, error) {
	if passwd == "" {
		return "", errors.New("password cannot empty")
	}

	result, err := bcrypt.GenerateFromPassword([]byte(passwd), bcryptCost())
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func ValidateHash(secret, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
	return err == nil
}
