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
cat > "${HOOK_FILE}" << 'EOF'
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Store list of originally staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

# Auto-format code
echo "Running go fmt..."
go fmt ./...

# Re-stage only the files that were originally staged
if [ -n "$STAGED_FILES" ]; then
    echo "$STAGED_FILES" | xargs -I {} git add {}
fi

# Run go vet
echo "Running go vet..."
go vet ./...

# Run unit tests (short mode, timeout 30s)
echo "Running unit tests..."
go test ./pkg/... -short -timeout 30s

echo "Pre-commit checks passed!"
EOF

# Make the hook executable
chmod +x "${HOOK_FILE}"

echo "Git pre-commit hook installed successfully!"
echo "The hook will run: auto-format, go vet, and unit tests before each commit."
