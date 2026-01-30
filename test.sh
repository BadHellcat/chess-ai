#!/bin/bash
# Test script for chess-ai project
# Make executable: chmod +x test.sh

set -e

echo "🧪 Running tests for chess-ai..."
echo ""

# Run tests
echo "📦 Testing all packages..."
go test ./... -v

echo ""
echo "✅ All tests passed!"
echo ""

# Run tests with coverage
echo "📊 Generating coverage report..."
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

echo ""
echo "🎉 Testing complete!"
