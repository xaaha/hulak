package envparser

import (
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestSubstituteVariables(t *testing.T) {
	varMap := make(map[string]string)
	varMap["varName"] = "replacedValue"
	varMap["secondName"] = "anju"
	varMap["thirdName"] = "pratik"
	varMap["anotherNumber"] = "5678"
	varMap["xaaha"] = "hero"
	varMap["number"] = "1234{{anotherNumber}}"

	unresolvedMessage := utils.UnResolvedVariable + "naa"
	testCases := []struct {
		expectedErrs   error
		stringToChange string
		expectedOutput string
	}{
		{
			stringToChange: "this/is/a/{{varName}}/with/{{number}}/{{xaaha}}",
			expectedOutput: "this/is/a/replacedValue/with/12345678/hero",
			expectedErrs:   nil,
		},
		{
			stringToChange: "a string without any curly braces",
			expectedOutput: "a string without any curly braces",
			expectedErrs:   nil,
		},
		{
			stringToChange: "1234 comes before {{naa}}",
			expectedOutput: "",
			expectedErrs:   utils.ColorError(unresolvedMessage),
		},
		{
			stringToChange: "",
			expectedOutput: "",
			expectedErrs:   utils.ColorError("variable string can't be empty"),
		},
	}
	for _, tc := range testCases {
		output, err := SubstitueVariables(tc.stringToChange, varMap)
		if output != tc.expectedOutput {
			t.Errorf(
				"Expected Output and does not match the result: \n%v \nvs \n%v",
				tc.expectedOutput,
				output,
			)
		}

		if err == nil && tc.expectedErrs == nil {
			// Both expected and got are nil, so this is correct; nothing to do here
			return
		} else if err == nil || tc.expectedErrs == nil {
			// One is nil and the other is not, so this is an error
			t.Errorf(
				"Expected or actual error is nil: expected %v, got %v",
				tc.expectedErrs,
				err,
			)
		} else if !strings.Contains(err.Error(), tc.expectedErrs.Error()) {
			// Both are non-nil, but do not match expected content
			t.Errorf(
				"Mismatch between expected vs actual err: \n%v \n%v",
				tc.expectedErrs,
				err,
			)
		}
	}
}
