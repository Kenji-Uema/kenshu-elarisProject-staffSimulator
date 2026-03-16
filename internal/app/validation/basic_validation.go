package validation

import (
	"reflect"
	"strings"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/validationErrors"
)

func (v *Validator) NotBlank(field, value string) *Validator {
	v.steps = append(v.steps, func() error {
		if strings.TrimSpace(value) == "" {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be blank"}
		}
		return nil
	})
	return v
}

func (v *Validator) NotZeroValue(field string, value any) *Validator {
	v.steps = append(v.steps, func() error {
		if reflect.ValueOf(value).IsZero() {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be zero value"}
		}
		return nil
	})
	return v
}

func (v *Validator) PositiveValue(field string, value int) *Validator {
	v.steps = append(v.steps, func() error {
		if value <= 0 {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must be greater than 0"}
		}
		return nil
	})

	return v
}
