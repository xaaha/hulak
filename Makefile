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

test-unit:
	@go test ./pkg/... -timeout 30s

# Format and vet
lint:
	@go fmt ./... && go vet ./...

# Quick check - runs lint and unit tests (no external API calls)
check:
	@echo "Running lint..."
	@go fmt ./...
	@go vet ./...
	@echo "Running unit tests..."
	@go test ./pkg/... -timeout 30s
	@echo "All checks passed!"

# Full check including e2e tests against real APIs
check-e2e:
	@make unit && make run && make run-all && make graphql

# Install git pre-commit hooks
install-hooks:
	@chmod +x scripts/install-hooks.sh && ./scripts/install-hooks.sh

bench:
	@go test -bench=. -benchmem ./... 2>&1 | grep '^Benchmark' | head -10

gen-coverage:
	@go test ./... -coverprofile=coverage.out

view-coverage:
	@go tool cover -html=coverage.out
