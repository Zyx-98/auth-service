package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/hatuan/auth-service/pkg/response"
)

var v = validator.New()

func Validate(data interface{}) []response.ValidationError {
	var errors []response.ValidationError

	err := v.Struct(data)
	if err == nil {
		return errors
	}

	for _, err := range err.(validator.ValidationErrors) {
		errors = append(errors, response.ValidationError{
			Field:   err.Field(),
			Message: getErrorMessage(err),
		})
	}

	return errors
}

func getErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "eqfield":
		return "Fields do not match"
	default:
		return "Invalid value"
	}
}
