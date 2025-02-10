package apicalls

// structure of the result to print in the console as the std output
type CustomResponse struct {
	Body           interface{} `json:"Body"`
	ResponseStatus string      `json:"Response Status"`
}
