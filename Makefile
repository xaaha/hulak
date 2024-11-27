run: 
	@go run . -fp "e2etests/test_collection/form_data.yaml" | jq .

run-all:
	@go run . -fp "e2etests/test_collection/url_encoded_form.yaml" -env prod | jq .

test:
	@go test ./...

build:
	@go build -o bin/hulak ./cmd/hulak/

