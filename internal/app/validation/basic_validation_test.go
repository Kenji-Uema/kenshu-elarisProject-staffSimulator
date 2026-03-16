package validation

import (
	"testing"
)

func TestValidator_NotBlank(t *testing.T) {
	testCases := map[string]struct {
		input          string
		isInputInvalid bool
	}{
		"blank string":        {"", true},
		"whitespace only":     {"   ", true},
		"non blank string":    {"value", false},
		"leading whitespace":  {" value", false},
		"trailing whitespace": {"value ", false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().NotBlank("field", test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %q, got %v", test.input, err)
			}
		})
	}
}

func TestValidator_NotZeroValue(t *testing.T) {
	type sample struct {
		value          any
		isInputInvalid bool
	}

	testCases := map[string]sample{
		"zero int":          {0, true},
		"non-zero int":      {1, false},
		"zero string":       {"", true},
		"zero value struct": {struct{ test int }{}, true},
		"non-zero string":   {"text", false},
		"nil slice":         {([]string)(nil), true},
		"non-nil slice":     {[]string{"one"}, false},
		"non zero struct":   {struct{ test int }{1}, false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().NotZeroValue("field", test.value).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %#v, got nil", test.value)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %#v, got %v", test.value, err)
			}
		})
	}
}

func TestValidator_PositiveValue(t *testing.T) {
	testCases := map[string]struct {
		input          int
		isInputInvalid bool
	}{
		"zero":     {0, true},
		"negative": {-5, true},
		"positive": {10, false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().PositiveValue("fieldName", test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %q, got nil", test.input)
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %v, got %v", test.input, err)
			}
		})
	}
}
