package validation

import (
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (v *Validator) NotNilObjectID(field string, id bson.ObjectID) *Validator {
	v.steps = append(v.steps, func() error {
		if id == bson.NilObjectID {
			return &validationErrors.ErrValidationConstrain{Field: field, Message: "must not be nil"}
		}
		return nil
	})
	return v
}

func (v *Validator) ValidCleaningStatus(field string, status string) *Validator {
	v.steps = append(v.steps, func() error {
		switch documents.CleaningStatus(status) {
		case documents.CleaningStatusPreparedForGuest,
			documents.CleaningStatusDailyCleaned,
			documents.CleaningStatusPreparedForSleep,
			documents.CleaningStatusFullyCleaned:
			return nil
		default:
			return &validationErrors.ErrValidationConstrain{
				Field:   field,
				Message: "must be one of prepared_for_guest, daily_cleaned, prepared_for_sleep, fully_cleaned",
			}
		}
	})
	return v
}
