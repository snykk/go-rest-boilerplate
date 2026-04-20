package validators

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

var mapHelepr = map[string]string{
	"required":       "is a required field",
	"email":          "is not a valid email address",
	"lowercase":      "must contain at least one lowercase letter",
	"uppercase":      "must contain at least one uppercase letter",
	"numeric":        "must contain at least one digit",
	"strongpassword": "must contain uppercase, lowercase, digit, and special character",
}

var needParam = []string{"min", "max", "containsany"}

// sharedValidate is reused across calls; validator.New() is relatively
// expensive and safe for concurrent use once constructed.
var (
	sharedValidate     *validator.Validate
	sharedValidateOnce sync.Once
)

func getValidator() *validator.Validate {
	sharedValidateOnce.Do(func() {
		sharedValidate = validator.New()
		_ = sharedValidate.RegisterValidation("strongpassword", StrongPassword)
	})
	return sharedValidate
}

func ValidatePayloads(payload interface{}) (err error) {
	var field, param, value, tag, message string

	err = getValidator().Struct(payload)
	if err != nil {
		var ve validator.ValidationErrors
		if !errors.As(err, &ve) {
			return err
		}
		for _, e := range ve {
			field = e.Field()
			tag = e.Tag()
			if s, ok := e.Value().(string); ok {
				value = s
			} else {
				value = ""
			}
			param = e.Param()

			if helpers.IsArrayContains(needParam, tag) {
				message = errWithParam(field, value, tag, param)
				continue
			}

			if value != "" {
				value = fmt.Sprintf("'%s' ", value)
			}
			if msg, ok := mapHelepr[tag]; ok {
				message = fmt.Sprintf("%s: %s%s", strings.ToLower(field), value, msg)
			} else {
				message = fmt.Sprintf("%s: %sfailed validation on %q", strings.ToLower(field), value, tag)
			}
		}

		return errors.New(message)
	}

	return nil
}

func errWithParam(field, value, tag, param string) string {
	var message string
	switch tag {
	case "min":
		message = fmt.Sprintf("must be at least %s characters long", param)
	case "max":
		message = fmt.Sprintf("must be less than %s characters", param)
	case "containsany":
		message = fmt.Sprintf("must contain at least one symbol of '%s'", param)
	}

	return fmt.Sprintf("%s: '%s' %s", field, value, message)
}
