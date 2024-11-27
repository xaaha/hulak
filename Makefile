run: 
	@go run .

run-all:
	@go run . -env prod

test:
	@go test ./...

# test:
# 	@go test ./path/
# 	@go test ./path2/

build:
	@go build -o bin/hulak ./cmd/hulak/

