package graphql

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
)

func GqlParser() {
	// schema.json is the response we get from the introspection query
	// readfile for now
	data, err := os.ReadFile("schema.json")
	if err != nil {
		fmt.Printf("Error reading schema.json: %v\n", err)
		return
	}

	// Parse introspection JSON directly
	var introspectionData introspection.Data
	if err := json.Unmarshal(data, &introspectionData); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Build a type map for easy lookup
	typeMap := make(map[string]*introspection.FullType)
	for _, t := range introspectionData.Schema.Types {
		typeMap[t.Name] = t
	}

	// Print Queries
	if introspectionData.Schema.QueryType.Name != "" {
		queryTypeName := introspectionData.Schema.QueryType.Name
		if queryType, ok := typeMap[queryTypeName]; ok {
			fmt.Println("=== QUERIES ===")
			for _, field := range queryType.Fields {
				desc := "No description"
				if field.Description != "" {
					desc = field.Description
				}
				fmt.Printf("  %s: %s\n", field.Name, desc)
			}
			fmt.Println()
		}
	}

	// Print Mutations
	if introspectionData.Schema.MutationType.Name != "" {
		mutationTypeName := introspectionData.Schema.MutationType.Name
		if mutationType, ok := typeMap[mutationTypeName]; ok {
			fmt.Println("=== MUTATIONS ===")
			for _, field := range mutationType.Fields {
				desc := "No description"
				if field.Description != "" {
					desc = field.Description
				}
				fmt.Printf("  %s: %s\n", field.Name, desc)
			}
			fmt.Println()
		}
	}
}
