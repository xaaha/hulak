run: 
	@go run . -fp "e2etests/test_collection/form_data.hk.yaml" 

run-all:
	@go run . -fp "e2etests/test_collection/url_encoded_form.hk.yaml" -env prod 

graphql:
	@go run . -fp "e2etests/test_collection/graphql.hk.yml"

auth2:
	@go run . -f "oAuth2" -env test

unit:
	@go test ./...

build:
	@go build -o . 

check:
	@make unit && make run && make run-all  && make graphql

bench:
	@go test -bench=. -benchmem ./... 2>&1 | grep '^Benchmark' | head -10


gen-coverage:
	@go test ./... -coverprofile=coverage.out

view-coverage:
	@go tool cover -html=coverage.out
