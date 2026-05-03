package validators

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

// FieldError describes a single failed validation rule. Returned in
// the API response so the client knows which field is wrong without
// having to parse a flat string.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidationErrors is the structured failure value returned by
// ValidatePayloads. It implements `error` so it composes naturally
// with the rest of the error-return idiom; handlers can type-assert
// and render the per-field detail.
type ValidationErrors struct {
	Errors []FieldError
}

func (v *ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(v.Errors))
	for _, e := range v.Errors {
		parts = append(parts, e.Field+": "+e.Message)
	}
	return strings.Join(parts, "; ")
}

var mapHelper = map[string]string{
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

// ValidatePayloads runs the struct's validate tags and returns nil
// on success. On failure it returns *ValidationErrors with one entry
// per failed field, so callers can render structured responses.
func ValidatePayloads(payload interface{}) error {
	if err := getValidator().Struct(payload); err != nil {
		var ve validator.ValidationErrors
		if !errors.As(err, &ve) {
			return err
		}
		out := &ValidationErrors{Errors: make([]FieldError, 0, len(ve))}
		for _, e := range ve {
			out.Errors = append(out.Errors, FieldError{
				Field:   strings.ToLower(e.Field()),
				Tag:     e.Tag(),
				Message: messageFor(e),
			})
		}
		return out
	}
	return nil
}

// messageFor produces the human-readable line used in both the flat
// .Error() string and the structured FieldError.Message.
func messageFor(e validator.FieldError) string {
	tag := e.Tag()
	param := e.Param()

	value := ""
	if s, ok := e.Value().(string); ok {
		value = s
	}

	if helpers.IsArrayContains(needParam, tag) {
		return paramMessage(value, tag, param)
	}
	if msg, ok := mapHelper[tag]; ok {
		if value != "" {
			return fmt.Sprintf("'%s' %s", value, msg)
		}
		return msg
	}
	return fmt.Sprintf("failed validation on %q", tag)
}

func paramMessage(value, tag, param string) string {
	switch tag {
	case "min":
		return fmt.Sprintf("must be at least %s characters long", param)
	case "max":
		return fmt.Sprintf("must be less than %s characters", param)
	case "containsany":
		return fmt.Sprintf("must contain at least one symbol of '%s'", param)
	}
	return fmt.Sprintf("failed %s validation", tag)
}
