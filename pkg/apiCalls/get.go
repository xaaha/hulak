package apicalls

import (
	"io"
	"log"
	"net/http"
)

// sample get call from https://blog.logrocket.com/making-http-requests-in-go/
func Get() {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/posts")
	if err != nil {
		log.Fatalln(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// Convert the body to type string
	sb := string(body)
	log.Print(sb)
}

// func Post(){
//   resp, err := http.Post("", 'x-www-form-urlencoded', body io.Reader)
// }
