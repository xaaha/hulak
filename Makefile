run: 
	@go run . -fp "e2etests/test_collection/form_data.yaml" 

# tests
run-all:
	@go run . -fp "e2etests/test_collection/url_encoded_form.yaml" -env prod 

graphql:
	@go run . -fp "e2etests/test_collection/graphql.yml"

auth2:
	@go run . -f "oAuth2" -env test

test:
	@go test ./...

build:
	@go build -o bin/hulak ./cmd/hulak/

check:
	@make test && make run && make run-all  && make graphql
