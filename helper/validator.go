package helper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	// Report errors using the json/form field name instead of the Go field name.
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form"} {
			name := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})

	return v
}

// ValidateStruct returns a list of readable validation errors, or nil when valid.
func ValidateStruct(s interface{}) []string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return []string{err.Error()}
	}

	var messages []string
	for _, e := range validationErrors {
		if e.Param() != "" {
			messages = append(messages, fmt.Sprintf("%s failed on '%s=%s' validation", e.Field(), e.Tag(), e.Param()))
		} else {
			messages = append(messages, fmt.Sprintf("%s failed on '%s' validation", e.Field(), e.Tag()))
		}
	}

	return messages
}
