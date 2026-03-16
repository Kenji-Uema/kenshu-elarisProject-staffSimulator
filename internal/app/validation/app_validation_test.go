package validation

import (
	"testing"
	"time"
)

func TestValidator_EmailFormat(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"without domain": {"test@test", true},
		"empty string":   {"", true},
		"valid format":   {"test@test.com", false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().EmailFormat(test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %s, got %v", test.input, err)
			}
		})
	}
}

func TestValidator_CleaningRequest(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"do not clean":   {"DO_NOT_CLEAN", true},
		"do not disturb": {"DO_NOT_DISTURB", false},
		"clean":          {"CLEAN", false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().CleaningRequest(test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %s, got %v", test.input, err)
			}
		})
	}
}

func TestValidator_Period(t *testing.T) {
	var now = time.Now()
	testCases := map[string]struct {
		start          time.Time
		end            time.Time
		isInputInvalid bool
	}{
		"start before end":     {now, now.AddDate(0, 0, 10), false},
		"start and end equals": {now, now, true},
		"start after end":      {now, now.AddDate(0, 0, -10), true},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().Period(test.start, test.end).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for start=%v end=%v, got nil", test.start, test.end)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for start=%v end=%v, got %v", test.start, test.end, err)
			}
		})
	}
}
