// Package apicalls has all things related to api call
package apicalls

// CustomResponse is structure of the result to print in the console as the std output
type CustomResponse struct {
	Body           any    `json:"Body"`
	ResponseStatus string `json:"Response Status"`
}
