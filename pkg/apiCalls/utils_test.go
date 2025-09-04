package apicalls

import (
	"maps"
	"testing"
)

func TestHandleUrlParams(t *testing.T) {
	tests := []struct {
		name              string
		apiInfoUrlParams  map[string]string
		expectedUrlParams map[string]string
	}{
		{
			name: "Params are passed",
			apiInfoUrlParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			expectedUrlParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		},
		{
			name:              "Params are not passed",
			apiInfoUrlParams:  nil,
			expectedUrlParams: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if !maps.Equal(tt.apiInfoUrlParams, tt.expectedUrlParams) {
				t.Errorf("HandleUrlParams = %v, expected %v", tt.apiInfoUrlParams, tt.expectedUrlParams)
			}
		})
	}
}
