package e2etests

import (
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
)

func RunFormData(envMap map[string]string) {
	apicalls.SendApiRequest(envMap, "e2etests/test_collection/form_data.yaml")
}

func RunFormDataError(envMap map[string]string) {
	apicalls.SendApiRequest(envMap, "e2etests/test_collection/form_data_error.yml")
}

func RunUrlEncodedFormData(envMap map[string]string) {
	apicalls.SendApiRequest(envMap, "e2etests/test_collection/url_encoded_form.yaml")
}
