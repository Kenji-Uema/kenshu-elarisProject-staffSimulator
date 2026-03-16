package validation

import (
	"testing"

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
