package apicalls

// HandleUrlParams handles setting url params if passed
// otherwise defaults to empty map
func HandleUrlParams(apiInfoUrlParams map[string]string) map[string]string {

	urlParams := map[string]string{}
	if len(apiInfoUrlParams) > 0 {
		urlParams = apiInfoUrlParams
	}

	return urlParams
}
