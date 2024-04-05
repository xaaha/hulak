run: 
	@go run cmd/hulak/main.go 

run-all:
	@cd cmd/hulak && go run . -env test

# test:
# 	@go test ./path/
# 	@go test ./path2/
