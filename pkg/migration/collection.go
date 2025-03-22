package migration

import (
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// KeyValuePair represents a generic key-value pair used in various Postman structures
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PMCollectionInfo represents the info object in a Postman collection
type PMCollectionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PMCOLLECTION struct {
	Info PMCollectionInfo `json:"info"`
	// TODO: ADD ITEM Structure after all the minute details are done
	// ITEM
}

// PMURL represents PMURL information in a request
type PMURL struct {
	Raw   string         `json:"raw"`
	Query []KeyValuePair `json:"query"`
}

// TODO: This can get big. Test/check all possible cases
type PMRequest struct {
	Description string                    `json:"description"`
	URL         PMURL                     `json:"url"`
	Method      yamlParser.HTTPMethodType `json:"method"`
	Header      []KeyValuePair            `json:"header"`
}

// CollectionItemRequest represents each request
type CollectionItemRequest struct {
	FileName string    `json:"name"`
	Request  PMRequest `json:"request"`
}

// CollectionItem represents an item in a Postman collection
type CollectionItem struct {
	Name    string                `json:"name"`
	Request CollectionItemRequest `json:"request"`
	// Response and events can be added here
}

// Collection represents a Postman 2.1 collection
type Collection struct {
	Info     PMCollectionInfo `json:"info"`
	Item     CollectionItem   `json:"item"`
	Variable []KeyValuePair   `json:"variable"`
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
