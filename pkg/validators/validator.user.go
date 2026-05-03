package validators

import (
	"unicode"

	"github.com/go-playground/validator/v10"
)

// StrongPassword enforces minimum password complexity: at least one
// uppercase letter, one lowercase letter, one digit, and one special
// character. Length is enforced separately via the "min" tag.
func StrongPassword(fl validator.FieldLevel) bool {
	password, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}
