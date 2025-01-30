package apicalls

import "io"

// struct for StandardCall
type ApiInfo struct {
	Body      io.Reader
	Headers   map[string]string
	UrlParams map[string]string
	Method    string
	Url       string
}

// structure of the result to print in the console as the std output
type CustomResponse struct {
	Body           interface{} `json:"Body"`
	ResponseStatus string      `json:"Response Status"`
}
