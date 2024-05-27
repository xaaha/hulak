run: 
	@go run cmd/hulak/main.go 

run-all:
	@cd cmd/hulak && go run . -env prod 

test:
	@go test ./...

# test:
# 	@go test ./path/
# 	@go test ./path2/

build:
	@go build -o bin/hulak ./cmd/hulak/

