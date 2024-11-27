package e2etests

import (
	"fmt"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func RunFormData(envMap map[string]string) {
	formDataJSONString := fileReader.ReadYamlForHttpRequest(
		"e2etests/test_collection/form_data.yaml",
		envMap,
	)
	apiInfo, err := apicalls.PrepareStruct(formDataJSONString)
	if err != nil {
		_ = fmt.Errorf("Error occoured in main %v", err)
	}
	resp := apicalls.StandardCall(apiInfo)
	fmt.Println(resp)
}

// runs actual form
func RunFormDataError(envMap map[string]string) {
	formDataJSONString := fileReader.ReadYamlForHttpRequest(
		"e2etests/test_collection/form_data_error.yml",
		envMap,
	)
	apiInfo, err := apicalls.PrepareStruct(formDataJSONString)
	if err != nil {
		_ = fmt.Errorf("Error occoured in main %v", err)
	}
	resp := apicalls.StandardCall(apiInfo)
	fmt.Println(resp)
}

// runs actual call from "test_collection/url_encoded_form.yaml.yaml"
func RunUrlEncodedFormData(envMap map[string]string) {
	formDataJSONString := fileReader.ReadYamlForHttpRequest(
		"e2etests/test_collection/url_encoded_form.yaml",
		envMap,
	)
	apiInfo, err := apicalls.PrepareStruct(formDataJSONString)
	if err != nil {
		_ = fmt.Errorf("Error occoured in main %v", err)
	}
	resp := apicalls.StandardCall(apiInfo)
	fmt.Println(resp)
}
