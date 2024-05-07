package envparser

import (
	"testing"
)

func TestTrimQuotes(t *testing.T) {
	// it's a bit tricky to test since trim quotes trims the quotest the value from env file
	testCases := []struct {
		input  string
		output string
	}{
		{input: "", output: ""},
		{input: "test's value", output: "test's value"},
		{input: "userNam2", output: "userNam2"},
	}

	for _, tc := range testCases {
		resultStr := trimQuotes(tc.input)
		if resultStr != tc.output {
			t.Errorf(
				"Expected output does not match the result: \n%v \nvs \n%v",
				tc.output,
				resultStr,
			)
		}
	}
}

func TestSetEnvironment(t *testing.T) {
	/* Test cases
	* provided flag is captured.
	* It captures only first argument
	* second argument should be ignored.
	 */
	// environmentFiles := []string{
	// 	"staging.env",
	// 	"test_GeneSis.env",
	// 	"proDuction-A7E.env",
	// }
}
