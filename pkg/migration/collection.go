package migration

import "fmt"

// KeyValuePair represents a generic key-value pair used in various Postman structures
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CollectionInfo represents the info object in a Postman collection
type CollectionInfo struct {
	PostmanID      string `json:"_postman_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Schema         string `json:"schema"`
	CollectionLink string `json:"_collection_link"`
}

// ItemRawUrl represents URL information in a request
type ItemRawUrl struct {
	Raw string `json:"raw"`
}

// CollectionItemRequest represents a request in a collection item
type CollectionItemRequest struct {
	Method string         `json:"method"`
	Header []KeyValuePair `json:"header"`
	URL    ItemRawUrl     `json:"url"`
}

// CollectionItem represents an item in a Postman collection
type CollectionItem struct {
	Name    string                `json:"name"`
	Request CollectionItemRequest `json:"request"`
	// Response and events can be added here
}

// Collection represents a Postman 2.1 collection
type Collection struct {
	Info     CollectionInfo `json:"info"`
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
	return fmt.Errorf("collection migration not yet implemented")
}
