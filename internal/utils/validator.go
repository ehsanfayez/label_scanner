package utils

import (
	"time"

	"github.com/go-playground/validator/v10"
)

type (
	ErrorResponse struct {
		Error       bool
		FailedField string
		Tag         string
		Value       interface{}
	}

	XValidator struct {
		Validator *validator.Validate
	}

	GlobalErrorHandlerResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

func Validate(data interface{}) []ErrorResponse {
	var validate = validator.New()
	_ = validate.RegisterValidation("DateTimeValidator", ValidateDateTime)
	validationErrors := []ErrorResponse{}

	errs := validate.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			// In this case data object is actually holding the User struct
			var elem ErrorResponse

			elem.FailedField = err.Field() // Export struct field name
			elem.Tag = err.Tag()           // Export struct tag
			elem.Value = err.Value()       // Export field value
			elem.Error = true

			validationErrors = append(validationErrors, elem)
		}
	}

	return validationErrors
}

func ValidationError(errs []ErrorResponse) map[string]string {
	errMsgs := make(map[string]string, 0)

	for _, err := range errs {
		if err.Tag == "DateTimeValidator" {
			errMsgs[CamelToSnake(err.FailedField)] = "The format of this field should follow this pattern '2006-01-02T15:04:05.999Z07:00'."
		} else {
			errMsgs[CamelToSnake(err.FailedField)] = "This field is " + err.Tag
		}

	}

	return errMsgs
}

func ValidateDateTime(fl validator.FieldLevel) bool {
	dateTimeStr := fl.Field().String()
	layout := time.RFC3339Nano

	// Attempt to parse the datetime string using the layout
	parsedTime, err := time.Parse(layout, dateTimeStr)
	if err != nil {
		return false // Parsing error
	}

	// Check if the day component is "00"
	if parsedTime.Day() == 0 {
		return false
	}

	return true
}
