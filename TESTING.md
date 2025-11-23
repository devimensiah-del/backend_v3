# Testing Guide for Backend v3

This guide explains how to run tests with different configurations.

## Quick Reference

```bash
# Using Make (recommended for Go developers)
make test              # Unit tests only (non-verbose)
make test-integration  # Integration tests only
make test-all          # All tests (verbose)
make test-coverage     # Generate coverage report

# Using npm (convenient for full-stack developers)
npm test                      # Unit tests only (non-verbose)
npm run test:integration      # Integration tests only
npm run test:all              # All tests
npm run test:coverage         # Generate coverage report
```

## Test Categories

### 1. **Unit Tests** (Fast, Isolated)
- Located in: `./domain/*/`, `./jobs/`
- Run with: `make test-unit` or `npm run test:unit`
- Characteristics:
  - No external dependencies
  - Use mocks for database, HTTP, storage
  - Fast execution (< 30s total)
  - Run on every commit

### 2. **Integration Tests** (Slower, Real Dependencies)
- Located in: `./integration_tests/`
- Run with: `make test-integration` or `npm run test:integration`
- Characteristics:
  - May require real services (Redis, PostgreSQL, APIs)
  - Slower execution (up to 60s)
  - Run before deployments

## Command Reference

### Makefile Commands (for Go developers)

| Command | Description | Verbose? | Timeout |
|---------|-------------|----------|---------|
| `make test` | Unit tests only (default) | No | 30s |
| `make test-unit` | Unit tests only | No | 30s |
| `make test-integration` | Integration tests only | No | 60s |
| `make test-verbose` | All tests | Yes | 30s |
| `make test-unit-verbose` | Unit tests | Yes | 30s |
| `make test-integration-verbose` | Integration tests | Yes | 60s |
| `make test-all` | All tests (unit + integration) | Yes | 60s |
| `make test-coverage` | All tests with coverage report | No | 60s |
| `make test-quick` | Quick unit tests only | No | 15s |
| `make test-domain PKG=<name>` | Test specific domain | Yes | 30s |
| `make test-failed` | Re-run known failing tests | Yes | 30s |
| `make clean` | Clean test cache and artifacts | - | - |

### NPM Scripts (for full-stack developers)

| Command | Description | Verbose? |
|---------|-------------|----------|
| `npm test` | Unit tests only (default) | No |
| `npm run test:unit` | Unit tests only | No |
| `npm run test:integration` | Integration tests only | No |
| `npm run test:all` | All tests (unit + integration) | No |
| `npm run test:verbose` | All tests (verbose) | Yes |
| `npm run test:unit:verbose` | Unit tests (verbose) | Yes |
| `npm run test:integration:verbose` | Integration tests (verbose) | Yes |
| `npm run test:coverage` | All tests with coverage HTML | No |
| `npm run test:quick` | Quick unit tests | No |
| `npm run test:failed` | Re-run known failing tests | Yes |
| `npm run clean` | Clean test cache | No |

## Examples

### Run Unit Tests Only (Non-Verbose)
```bash
make test-unit
# or
npm run test:unit
```

**Output:**
```
🧪 Running unit tests (non-verbose)...
ok      backend_v3/domain/submission    2.115s
ok      backend_v3/domain/enrichment    7.705s
ok      backend_v3/jobs                 2.066s
```

### Run Unit Tests with Verbose Output
```bash
make test-unit-verbose
# or
npm run test:unit:verbose
```

**Output:**
```
🧪 Running unit tests (verbose)...
=== RUN   TestService_Create
=== RUN   TestService_Create/success_-_creates_submission
--- PASS: TestService_Create/success_-_creates_submission (0.00s)
...
```

### Run Integration Tests Only
```bash
make test-integration
# or
npm run test:integration
```

### Run All Tests (Unit + Integration)
```bash
make test-all
# or
npm run test:all
```

### Generate Coverage Report
```bash
make test-coverage
# or
npm run test:coverage
```

This generates `coverage.html` that you can open in your browser.

### Test a Specific Domain
```bash
make test-domain PKG=submission
# Tests only ./domain/submission/
```

### Re-run Failed Tests
```bash
make test-failed
# or
npm run test:failed
```

This re-runs only the tests that are currently failing.

## Verbose vs Non-Verbose

### Non-Verbose Output (default for quick feedback)
```
ok      backend_v3/domain/submission    2.115s
ok      backend_v3/domain/enrichment    7.705s
FAIL    backend_v3/domain/analysis      2.542s
```

### Verbose Output (for debugging)
```
=== RUN   TestService_Create
=== RUN   TestService_Create/success
--- PASS: TestService_Create/success (0.00s)
=== RUN   TestService_Create/validation_error
--- FAIL: TestService_Create/validation_error (0.00s)
    service_test.go:45: Expected error, got nil
```

## Test Naming Conventions

All tests follow this pattern:
- `Test<Component>_<Scenario>` - e.g., `TestService_Create_Success`
- Integration tests are in `./integration_tests/` directory
- Unit tests use `_test.go` suffix in the same package

## Common Workflows

### During Development (Fast Feedback)
```bash
make test-quick
# Runs only unit tests with short timeout
```

### Before Committing (Local Validation)
```bash
make test-unit
# Runs all unit tests
```

### Before Deploying (Full Validation)
```bash
make test-all
# Runs unit + integration tests
```

### Investigating Failures
```bash
make test-unit-verbose
# Shows detailed test output
```

### Continuous Integration
```bash
make test-coverage
# Runs all tests and generates coverage report for CI
```

## Troubleshooting

### Tests are slow
- Use `make test-quick` for faster feedback
- Run only the domain you're working on: `make test-domain PKG=submission`

### Tests are failing
- Run with verbose: `make test-verbose`
- Check if it's a known failure: `make test-failed`
- Clean test cache: `make clean` then re-run

### Coverage report not generating
- Ensure you have write permissions
- Run: `make clean` then `make test-coverage`

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run Unit Tests
  run: make test-unit

- name: Run Integration Tests
  run: make test-integration

- name: Generate Coverage
  run: make test-coverage

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Test Fixtures](./integration_tests/fixtures/)
