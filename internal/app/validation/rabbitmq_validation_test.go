package validation

import "testing"

func TestValidator_ExchangeName(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"valid format":         {"ex.my-exchange_1", false},
		"missing prefix":       {"my-exchange", true},
		"empty string":         {"", true},
		"invalid characters":   {"ex.*bad", true},
		"double prefix":        {"ex.ex-second", false},
		"underscore only name": {"ex._", false},
		"dash only name":       {"ex.--", false},
		"spaces not allowed":   {"ex.my exchange", true},
		"trailing dot":         {"ex.", true},
		"prefix only":          {"ex", true},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().ExchangeName(test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %q, got %v", test.input, err)
			}
		})
	}
}

func TestValidator_AllowedContentType(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"allowed protobuf":   {"application/protobuf", false},
		"json rejected":      {"application/json", true},
		"uppercase protobuf": {"APPLICATION/PROTOBUF", true},
		"text plain":         {"text/plain", true},
		"empty string":       {"", true},
		"multipart":          {"multipart/form-data", true},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().AllowedContentType(test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %q, got %v", test.input, err)
			}
		})
	}
}
