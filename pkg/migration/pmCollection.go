package migration

import (
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// overall json structure for 2.1
type PmCollection struct {
	Info Info `json:"info"`
	Item Item `json:"name"`
}

// Info represents the info object in a Postman collection
type Info struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// when collection contains subfolders folders
type Item struct {
	Item []RequestFile `json:"item"`
}

type EachItemObject struct {
	Name        string      `json:"name"`
	Item        Item        `json:"item"` // present if nested directories exists
	RequestFile RequestFile `json:"request"`
}

// PmUrl represents PmUrl information in a request
type PmUrl struct {
	Raw   string         `json:"raw"`
	Query []KeyValuePair `json:"query"`
}

// TODO: Add all different type of body mode coming from postman
// formdata, urlencoded, graphql (query and variable), and more
type BodyMode struct {
	Raw string `json:"raw"`
}

// TODO: Add more as above
type PmBody struct {
	Mode BodyMode `json:"mode"`
	Raw  string   `json:"raw"`
}

// TODO: This can get big. Test/check all possible cases
// Represents individual request file
type RequestFile struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	URL         PmUrl                     `json:"url"`
	Method      yamlParser.HTTPMethodType `json:"method"`
	Header      []KeyValuePair            `json:"header"`
	Body        PmBody                    `json:"body"`
}

// CollectionItemRequest represents each request
type CollectionItemRequest struct {
	FileName string      `json:"name"`
	Request  RequestFile `json:"request"`
}

// CollectionItem represents an item in a Postman collection
type CollectionItem struct {
	Name    string                `json:"name"`
	Request CollectionItemRequest `json:"request"`
	// Response and events can be added here
}

// Collection represents a Postman 2.1 collection
type Collection struct {
	Info     Info           `json:"info"`
	Item     CollectionItem `json:"item"`
	Variable []KeyValuePair `json:"variable"`
}

// IsCollection determines if the JSON contains a Postman collection
func IsCollection(jsonString map[string]any) bool {
	_, infoExists := jsonString["info"]
	_, itemExists := jsonString["item"]
	return infoExists && itemExists
}

// MigrateCollection migrates a Postman collection to the desired format
// To be implemented
func MigrateCollection(collection Collection) error {
	// Implementation to come
	return utils.ColorError("collection migration not yet implemented")
}
