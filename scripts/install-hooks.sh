#!/bin/bash
# Install git pre-commit hook for Hulak project
# This hook auto-formats code, runs vet, and unit tests before commits

set -e

HOOKS_DIR=".git/hooks"
HOOK_FILE="${HOOKS_DIR}/pre-commit"

# Check if we're in a git repository
if [ ! -d ".git" ]; then
	echo "Error: Not in a git repository root directory"
	exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "${HOOKS_DIR}"

# Create the pre-commit hook
cat >"${HOOK_FILE}" <<'EOF'
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Collect staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM -- '*.go')

if [ -z "$STAGED_GO_FILES" ]; then
    echo "No staged .go files, skipping Go checks."
    exit 0
fi

# Build list of unique package directories from staged files
STAGED_PKGS=$(echo "$STAGED_GO_FILES" | xargs -I {} dirname {} | sort -u | sed 's|^|./|')

# Auto-format only staged Go files
echo "Running go fmt on staged files..."
echo "$STAGED_GO_FILES" | xargs gofmt -w

# Re-stage formatted files
echo "$STAGED_GO_FILES" | xargs git add

# Lint only affected packages
echo "Running golangci-lint on staged packages..."
echo "$STAGED_PKGS" | xargs golangci-lint run

# Vet only affected packages
echo "Running go vet on staged packages..."
echo "$STAGED_PKGS" | xargs go vet

# Run unit tests (short mode, timeout 30s)
echo "Running unit tests..."
go test ./pkg/... -short -timeout 30s

echo "Pre-commit checks passed!"
EOF

# Make the hook executable
chmod +x "${HOOK_FILE}"

echo "Git pre-commit hook installed successfully!"
echo "The hook will run: auto-format, go vet, and unit tests before each commit."
