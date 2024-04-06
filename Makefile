run: 
	@go run cmd/hulak/main.go 

run-all:
	@cd cmd/hulak && go run . -env prod 

# test:
# 	@go test ./path/
# 	@go test ./path2/
