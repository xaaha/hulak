// Package features have all the additional features hulak supports
package features

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IntrospectionResponse  Root structure for the GraphQL introspection response
type IntrospectionResponse struct {
	Response struct {
		StatusCode int `json:"status_code"`
		Body       struct {
			Data struct {
				Schema Schema `json:"__schema"`
			} `json:"data"`
			Duration string `json:"duration"`
		} `json:"body"`
	} `json:"response"`
}

// Schema represents the GraphQL schema
type Schema struct {
	Types            []Type   `json:"types"`
	QueryType        TypeRef  `json:"queryType"`
	MutationType     TypeRef  `json:"mutationType"`
	SubscriptionType *TypeRef `json:"subscriptionType,omitempty"`
}

// Type represents a GraphQL type
type Type struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
	InputFields []Field `json:"inputFields"`
}

// Field represents a field in a GraphQL type
type Field struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Args        []Arg   `json:"args"`
	Type        TypeRef `json:"type"`
}

// Arg represents an argument to a field
type Arg struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        TypeRef `json:"type"`
}

// TypeRef represents a type reference
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name,omitempty"`
	OfType *TypeRef `json:"ofType,omitempty"`
}

func main() {
	// Read the JSON file
	jsonFile := os.Args[1]
	if jsonFile == "" {
		fmt.Println("Please provide a JSON file path as argument")
		os.Exit(1)
	}

	content, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse the JSON
	var response IntrospectionResponse
	err = json.Unmarshal(content, &response)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	schema := response.Response.Body.Data.Schema
	outDir := "graphql_operations"

	// Create output directory
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate operation files
	generateOperations(schema, outDir)
}

func generateOperations(schema Schema, outDir string) {
	// Find the root types
	queryType := findTypeByName(schema.Types, schema.QueryType.Name)

	var mutationType *Type
	if schema.MutationType.Name != "" {
		mutationType = findTypeByName(schema.Types, schema.MutationType.Name)
	}

	var subscriptionType *Type
	if schema.SubscriptionType != nil && schema.SubscriptionType.Name != "" {
		subscriptionType = findTypeByName(schema.Types, schema.SubscriptionType.Name)
	}

	// Generate queries file
	if queryType != nil {
		generateOperationFile(queryType, schema.Types, "queries", outDir)
		fmt.Println("Generated queries file")
	}

	// Generate mutations file
	if mutationType != nil {
		generateOperationFile(mutationType, schema.Types, "mutations", outDir)
		fmt.Println("Generated mutations file")
	}

	// Generate subscriptions file
	if subscriptionType != nil {
		generateOperationFile(subscriptionType, schema.Types, "subscriptions", outDir)
		fmt.Println("Generated subscriptions file")
	}
}

func findTypeByName(types []Type, name string) *Type {
	for i := range types {
		if types[i].Name == name {
			return &types[i]
		}
	}
	return nil
}

func generateOperationFile(rootType *Type, allTypes []Type, opType string, outDir string) {
	var sb strings.Builder

	titleCaser := cases.Title(language.English)
	sb.WriteString(fmt.Sprintf("# Generated %s operations\n\n", titleCaser.String(opType)))

	prefix := ""
	switch opType {
	case "queries":
		prefix = "query"
	case "mutations":
		prefix = "mutation"
	case "subscriptions":
		prefix = "subscription"
	}

	// Skip built-in types and empty-field types
	if rootType.Fields == nil {
		return
	}

	// Process each field in the root type
	for _, field := range rootType.Fields {
		if field.Name == "__schema" || field.Name == "__type" ||
			strings.HasPrefix(field.Name, "__") {
			continue // Skip introspection fields
		}

		opName := fieldNameToCamelCase(field.Name)
		sb.WriteString(fmt.Sprintf("%s %s", prefix, opName))

		// Add arguments
		if len(field.Args) > 0 {
			sb.WriteString("(")
			for i, arg := range field.Args {
				if i > 0 {
					sb.WriteString(", ")
				}
				argType := getTypeString(arg.Type)
				sb.WriteString(fmt.Sprintf("$%s: %s", arg.Name, argType))
			}
			sb.WriteString(")")
		}

		sb.WriteString(" {\n")
		sb.WriteString(fmt.Sprintf("  %s", field.Name))

		// Add arguments to field call
		if len(field.Args) > 0 {
			sb.WriteString("(")
			for i, arg := range field.Args {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s: $%s", arg.Name, arg.Name))
			}
			sb.WriteString(")")
		}

		sb.WriteString(" {\n")

		// Add selection set
		outputType := unwrapType(field.Type)
		typeData := findTypeByName(allTypes, outputType)
		if typeData != nil && len(typeData.Fields) > 0 {
			generateSelectionSet(typeData, allTypes, 4, &sb, make(map[string]bool))
		} else {
			sb.WriteString("    # This type has no fields\n")
		}

		sb.WriteString("  }\n")
		sb.WriteString("}\n\n")
	}

	// Write the file
	filename := filepath.Join(outDir, fmt.Sprintf("%s.graphql", opType))
	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing %s file: %v\n", opType, err)
	}
}

func generateSelectionSet(
	typeData *Type,
	allTypes []Type,
	indent int,
	sb *strings.Builder,
	visited map[string]bool,
) {
	// Avoid infinite recursion
	if visited[typeData.Name] {
		return
	}

	// For complex object types, don't go too deep
	if typeData.Kind == "OBJECT" && indent > 8 {
		sb.WriteString(strings.Repeat(" ", indent) + "# Further fields omitted for brevity\n")
		return
	}

	// Only add fields for object types
	if typeData.Kind != "OBJECT" && typeData.Kind != "INTERFACE" && typeData.Kind != "UNION" {
		return
	}

	// Mark as visited
	visited[typeData.Name] = true

	// Add fields
	for _, field := range typeData.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue // Skip introspection fields
		}

		sb.WriteString(strings.Repeat(" ", indent) + field.Name)

		// Add arguments for field if any
		if len(field.Args) > 0 {
			// For simplicity we're skipping arguments at deeper levels
			sb.WriteString(" # Has arguments, customize as needed")
		}

		// Get the output type
		outputType := unwrapType(field.Type)
		fieldTypeData := findTypeByName(allTypes, outputType)

		// If it's an object type, add nested fields
		if fieldTypeData != nil &&
			(fieldTypeData.Kind == "OBJECT" || fieldTypeData.Kind == "INTERFACE" || fieldTypeData.Kind == "UNION") &&
			len(fieldTypeData.Fields) > 0 {
			sb.WriteString(" {\n")

			// Make a copy of the visited map to avoid affecting sibling fields
			nestedVisited := make(map[string]bool)
			maps.Copy(nestedVisited, visited)

			generateSelectionSet(fieldTypeData, allTypes, indent+2, sb, nestedVisited)
			sb.WriteString(strings.Repeat(" ", indent) + "}")
		}

		sb.WriteString("\n")
	}

	// Unmark as visited when backtracking
	visited[typeData.Name] = false
}

func unwrapType(typeRef TypeRef) string {
	// Navigate through wrappers (NON_NULL, LIST) to get the actual type name
	if typeRef.Name != "" {
		return typeRef.Name
	}

	if typeRef.OfType != nil {
		return unwrapType(*typeRef.OfType)
	}

	return "Unknown"
}

func getTypeString(typeRef TypeRef) string {
	if typeRef.Kind == "NON_NULL" {
		if typeRef.OfType != nil {
			return getTypeString(*typeRef.OfType) + "!"
		}
		return "Unknown!"
	}

	if typeRef.Kind == "LIST" {
		if typeRef.OfType != nil {
			return "[" + getTypeString(*typeRef.OfType) + "]"
		}
		return "[Unknown]"
	}

	return typeRef.Name
}

func fieldNameToCamelCase(name string) string {
	words := strings.Split(name, "_")
	titleCaser := cases.Title(language.English)
	for i := range words {
		words[i] = titleCaser.String(words[i])
	}
	return strings.Join(words, "")
}
