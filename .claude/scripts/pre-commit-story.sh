#!/bin/bash

# Pre-commit checks for story completion
# Runs: gofmt, go vet, go build, go test with coverage

set -euo pipefail

echo "=== Pre-commit Story Checks ==="
echo ""

# Check if there are any .go files
if ! find . -name "*.go" -not -path "./vendor/*" | grep -q .; then
    echo "No Go files found, skipping Go checks"
    exit 0
fi

# 1. Format check
echo "1. Checking gofmt..."
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor/ || true)
if [[ -n "$UNFORMATTED" ]]; then
    echo "   FAIL: Unformatted files:"
    echo "$UNFORMATTED" | sed 's/^/   - /'
    echo ""
    echo "   Run: gofmt -w ."
    exit 1
fi
echo "   OK"
echo ""

# 2. Vet check
echo "2. Running go vet..."
if ! go vet ./... 2>&1; then
    echo "   FAIL: go vet found issues"
    exit 1
fi
echo "   OK"
echo ""

# 3. Build check
echo "3. Checking go build..."
if ! go build ./... 2>&1; then
    echo "   FAIL: Build failed"
    exit 1
fi
echo "   OK"
echo ""

# 4. Test with coverage
echo "4. Running tests with coverage..."
if ! go test -coverprofile=coverage.out ./... 2>&1; then
    echo "   FAIL: Tests failed"
    rm -f coverage.out
    exit 1
fi

# 5. Coverage threshold check
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
rm -f coverage.out

echo "   Coverage: ${COVERAGE}%"
if (( $(echo "$COVERAGE < 90" | bc -l) )); then
    echo "   FAIL: Coverage ${COVERAGE}% is below 90% threshold"
    exit 1
fi
echo "   OK"
echo ""

echo "=== All checks passed ==="
