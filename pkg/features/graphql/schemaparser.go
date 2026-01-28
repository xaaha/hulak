package graphql

import (
	"fmt"
	"os"

	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

func GqlParser() {
	data, err := os.ReadFile("schema.graphql")
	if err != nil {
		fmt.Printf("Error reading schema.graphql: %v\n", err)
		return
	}

	// Parse the schema using gqlparser
	schema, parseErr := gqlparser.LoadSchema(&ast.Source{
		Name:  "schema.graphql",
		Input: string(data),
	})
	if parseErr != nil {
		fmt.Printf("Error parsing schema: %v\n", parseErr)
		return
	}

	// Print Queries
	if schema.Query != nil {
		fmt.Println("=== QUERIES ===")
		for _, field := range schema.Query.Fields {
			desc := "No description"
			if field.Description != "" {
				desc = field.Description
			}
			fmt.Printf("  %s: %s\n", field.Name, desc)
		}
		fmt.Println()
	}

	// Print Mutations
	if schema.Mutation != nil {
		fmt.Println("=== MUTATIONS ===")
		for _, field := range schema.Mutation.Fields {
			desc := "No description"
			if field.Description != "" {
				desc = field.Description
			}
			fmt.Printf("  %s: %s\n", field.Name, desc)
		}
		fmt.Println()
	}

	// Print Subscriptions
	if schema.Subscription != nil {
		fmt.Println("=== SUBSCRIPTIONS ===")
		for _, field := range schema.Subscription.Fields {
			desc := "No description"
			if field.Description != "" {
				desc = field.Description
			}
			fmt.Printf("  %s: %s\n", field.Name, desc)
		}
		fmt.Println()
	}

	// Print all types
	fmt.Println("=== ALL TYPES ===")
	for name, typeDef := range schema.Types {
		// Skip built-in types (they start with __)
		if len(name) > 2 && name[:2] == "__" {
			continue
		}
		desc := ""
		if typeDef.Description != "" {
			desc = fmt.Sprintf(" - %s", typeDef.Description)
		}
		fmt.Printf("  %s (%s)%s\n", name, typeDef.Kind, desc)
	}
}
