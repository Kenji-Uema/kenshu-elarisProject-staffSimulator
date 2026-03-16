package validation

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/validationErrors"
)

var exchangeNameRe = regexp.MustCompile(`^ex\.[A-Za-z0-9_-]+$`)
var allowedContentTypes = []string{"application/json"}

func (v *Validator) ExchangeName(name string) *Validator {
	v.steps = append(v.steps, func() error {
		if !exchangeNameRe.MatchString(name) {
			return &validationErrors.ErrValidationConstrain{
				Field:   "exchangeName",
				Message: "must be in format ex.<name of exchange>",
			}
		}
		return nil
	})
	return v
}

func (v *Validator) AllowedContentType(contentType string) *Validator {
	v.steps = append(v.steps, func() error {
		if !slices.Contains(allowedContentTypes, contentType) {
			return &validationErrors.ErrValidationConstrain{
				Field:   "contentType",
				Message: fmt.Sprintf("must be of those: %q", allowedContentTypes),
			}
		}
		return nil
	})

	return v
}
