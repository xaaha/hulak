run: 
	@go run . -fp "e2etests/test_collection/form_data.yaml" | jq .

# tests
run-all:
	@go run . -fp "e2etests/test_collection/url_encoded_form.yaml" -env prod | jq .

graphql:
	@go run . -fp "e2etests/test_collection/graphql.yml"

test:
	@go test ./...

build:
	@go build -o bin/hulak ./cmd/hulak/

