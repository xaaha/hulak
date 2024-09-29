package hulak_yaml_reader

import (
	"fmt"
	"log"
	"os"

	yaml "github.com/goccy/go-yaml"
)

type User struct {
	Name  string `yaml:"name"`
	Age   string `yaml:"age"`
	Email string `yaml:"email"`
}

func ReadingYamlWithStruct() {
	file, err := os.Open("test_collection/user.yaml")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	var user User
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&user); err != nil {
		log.Fatalf("error decoding data: %v", err)
	}
	fmt.Printf("Name: %s, Age: %v, Email: %s", user.Name, user.Age, user.Email)
}

// simply import
// reader.SampleReader()
