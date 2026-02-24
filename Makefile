bench:
	@go test -bench=. -benchmem ./... 2>&1 | grep '^Benchmark' | head -10

gen-coverage:
	@go test ./... -coverprofile=coverage.out

view-coverage:
	@go tool cover -html=coverage.out
