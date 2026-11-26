// Package validate checks struct fields against their validate tags and reports the failures without ever
// repeating the submitted values.
package validate

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError is one field that failed one rule.
type FieldError struct {
	// Field is the field's JSON name, such as "title".
	Field string `json:"field"`
	// Rule is the validate tag that failed, such as "required".
	Rule string `json:"rule"`
	// Message describes the rule; it never contains the submitted value.
	Message string `json:"message"`
} // @name FieldError

// Errors lists every field that failed validation, in field order.
type Errors []FieldError

// Error joins the fields and their messages, as in "title: is required; slug: must be at most 5 characters".
func (e Errors) Error() string {
	parts := make([]string, len(e))
	for i, fieldError := range e {
		parts[i] = fieldError.Field + ": " + fieldError.Message
	}
	return strings.Join(parts, "; ")
}

var engine = newEngine()

func newEngine() *validator.Validate {
	engine := validator.New(validator.WithRequiredStructEnabled())
	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
	return engine
}

// Struct checks value against its validate tags. It returns nil when value is valid, Errors when one or more fields
// are invalid, and another error when value cannot be checked, for example because it is not a struct.
func Struct(value any) error {
	err := engine.Struct(value)
	if err == nil {
		return nil
	}

	var failed validator.ValidationErrors
	if !errors.As(err, &failed) {
		return err
	}

	fields := make(Errors, len(failed))
	for i, fieldError := range failed {
		fields[i] = FieldError{Field: fieldError.Field(), Rule: fieldError.Tag(), Message: message(fieldError)}
	}
	return fields
}

func message(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "max":
		return bound("at most", fieldError)
	case "min":
		return bound("at least", fieldError)
	case "oneof":
		return "must be one of: " + fieldError.Param()
	case "uuid":
		return "must be a valid UUID"
	case "email":
		return "must be a valid email address"
	}
	return "is invalid"
}

func bound(limit string, fieldError validator.FieldError) string {
	if fieldError.Kind() == reflect.String {
		return "must be " + limit + " " + fieldError.Param() + " characters"
	}
	return "must be " + limit + " " + fieldError.Param()
}
