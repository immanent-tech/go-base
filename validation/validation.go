// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package validation

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

var ErrNilObject = errors.New("object is nil")
var ErrInvalid = errors.New("object is invalid")

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
}

// Error is a map of fields and their validation errors.
type Error struct {
	Details string
	fields  map[string]string
}

func (e *Error) Error() string {
	return "invalid data: " + e.Details
}

// IsValid will check if an object is valid according to the validation tags on the object. It does not return any
// details of validation issues, only a boolean for valid (true) or invalid (false).
func IsValid[T any](obj T) bool {
	err := Validate.Struct(obj)
	return err != nil
}

// ParseStructValidationErrors takes the underlying validation errors and formats them so that each struct field has an
// array of all validation errors associated with it.
func ParseStructValidationErrors(err error) error {
	if validateErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		fields := make(map[string]string)
		// Generate details of fields that failed validation.
		var details strings.Builder
		for fieldErr := range slices.Values(validateErrs) {
			fmt.Fprintf(&details, "field=%s tag=%s value=%v", fieldErr.Field(), fieldErr.Tag(), fieldErr.Value())
			fields[fieldErr.Field()] = fieldErr.Error()
		}
		return &Error{
			Details: details.String(),
			fields:  fields,
		}
	}
	return err
}
