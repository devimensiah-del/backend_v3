#!/bin/bash
# Integration Tests Runner
# This script runs the integration tests with proper configuration

set -e

echo "========================================="
echo "  Imensiah Integration Tests Runner"
echo "========================================="
echo ""

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "⚠️  DATABASE_URL not found in environment"
    echo "   Tests will use mock mode (limited functionality)"
    echo ""
else
    echo "✓ DATABASE_URL found"
fi

# Check for OpenRouter API key
if [ -z "$OPENROUTER_API_KEY" ] || [ "$OPENROUTER_API_KEY" == "sk-or-your-openrouter-key" ]; then
    echo "⚠️  OPENROUTER_API_KEY not found or is default"
    echo "   LLM tests will use mock responses"
    echo ""
else
    echo "✓ OPENROUTER_API_KEY found (using real LLM)"
fi

# Check for Supabase URL
if [ -z "$SUPABASE_URL" ] || [ "$SUPABASE_URL" == "https://your-project-id.supabase.co" ]; then
    echo "⚠️  SUPABASE_URL not found or is default"
    echo "   Storage tests will use mock in-memory storage"
    echo ""
else
    echo "✓ SUPABASE_URL found (using real Supabase storage)"
fi

echo "========================================="
echo ""

# Run tests
echo "Running integration tests..."
echo ""

if [ "$1" == "-v" ] || [ "$1" == "--verbose" ]; then
    go test -v -timeout 5m
elif [ "$1" == "-bench" ] || [ "$1" == "--benchmark" ]; then
    go test -bench=. -benchmem
elif [ "$1" == "-short" ]; then
    go test -short -v
else
    go test -timeout 5m
fi

EXIT_CODE=$?

echo ""
echo "========================================="
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Some tests failed (exit code: $EXIT_CODE)"
fi
echo "========================================="

exit $EXIT_CODE
