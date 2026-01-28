package graphql

import (
	"fmt"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astvalidation"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// ValidateQuery validates the query user provides
// It's just an ai prototype right now
func ValidateQuery(schemaSDL, query string) error {
	schemaDoc, report := astparser.ParseGraphqlDocumentString(schemaSDL)
	if report.HasErrors() {
		return fmt.Errorf("schema error: %s", report.Error())
	}

	queryDoc, report := astparser.ParseGraphqlDocumentString(query)
	if report.HasErrors() {
		return fmt.Errorf("query parse error: %s", report.Error())
	}

	validator := astvalidation.DefaultOperationValidator()
	var validationReport operationreport.Report
	validator.Validate(&queryDoc, &schemaDoc, &validationReport)

	if validationReport.HasErrors() {
		return fmt.Errorf("validation error: %s", validationReport.Error())
	}
	return nil
}
