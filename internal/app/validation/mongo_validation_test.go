package validation

import (
	"testing"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestValidator_NotNilObjectID(t *testing.T) {
	testCases := map[string]struct {
		input          bson.ObjectID
		isInputInvalid bool
	}{
		"nil object id":  {bson.NilObjectID, true},
		"non-nil object": {bson.NewObjectID(), false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().NotNilObjectID("id", test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %s, got nil", test.input.Hex())
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %s, got %v", test.input.Hex(), err)
			}
		})
	}
}

func TestValidator_ValidCleaningStatus(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"prepared for guest": {string(documents.CleaningStatusPreparedForGuest), false},
		"daily cleaned":      {string(documents.CleaningStatusDailyCleaned), false},
		"prepared for sleep": {string(documents.CleaningStatusPreparedForSleep), false},
		"fully cleaned":      {string(documents.CleaningStatusFullyCleaned), false},
		"invalid status":     {"INVALID", true},
		"blank status":       {"", true},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().ValidCleaningStatus("cleaningStatus", test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %q, got %v", test.input, err)
			}
		})
	}
}
