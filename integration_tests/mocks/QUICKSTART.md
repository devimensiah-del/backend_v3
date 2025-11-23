# Supabase Mock - Quick Start Guide

## 🚀 Get Started in 60 Seconds

### 1. Import the Mock

```go
import "your-project/backend_v3/integration_tests/mocks"
```

### 2. Write Your First Test

```go
package mypackage_test

import (
    "context"
    "testing"
    "your-project/backend_v3/integration_tests/mocks"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyFirstMock(t *testing.T) {
    // Create mock
    mock := mocks.NewSupabaseJSONMockDefault()
    defer mock.Clear()

    ctx := context.Background()

    // Upload a file
    content := []byte("My first test file")
    url, err := mock.Upload(ctx, "test/file.pdf", content, "application/pdf")

    // Verify
    require.NoError(t, err)
    assert.Contains(t, url, "test/file.pdf")

    // Retrieve
    data, exists := mock.GetFileData("test/file.pdf")
    require.True(t, exists)
    assert.Equal(t, content, data)
}
```

### 3. Run Your Test

```bash
cd backend_v3
go test ./integration_tests/mocks -v -run TestMyFirstMock
```

## 📦 Common Use Cases

### Upload and Verify

```go
mock := mocks.NewSupabaseJSONMockDefault()

// Upload
url, _ := mock.Upload(ctx, "path/file.pdf", data, "application/pdf")

// Verify file exists
file, exists := mock.GetFile("path/file.pdf")
fmt.Printf("File size: %d bytes\n", file.Size)
```

### Multiple Files

```go
mock := mocks.NewSupabaseJSONMockDefault()

files := map[string][]byte{
    "report-1.pdf": []byte("Report 1"),
    "report-2.pdf": []byte("Report 2"),
}

for path, content := range files {
    mock.Upload(ctx, path, content, "application/pdf")
}

fmt.Printf("Total files: %d\n", mock.FileCount())
```

### Get Metadata Only

```go
metadata, exists := mock.GetFileMetadata("path/file.pdf")
if exists {
    fmt.Printf("Size: %v\n", metadata["size"])
    fmt.Printf("Type: %v\n", metadata["content_type"])
    fmt.Printf("URL: %v\n", metadata["public_url"])
}
```

### Load from JSON Configuration

```go
// With custom test scenarios and pre-loaded files
mock, err := mocks.NewSupabaseJSONMock("path/to/supabase_mock_data.json")
if err != nil {
    t.Fatal(err)
}

// Access pre-loaded files
file, exists := mock.GetFile("test-submission-123/report-v1.pdf")
```

### Use Test Scenarios

```go
mock, _ := mocks.NewSupabaseJSONMock("supabase_mock_data.json")

scenario, _ := mock.GetTestScenario("successful_upload")
path := scenario.Input["path"].(string)

// Run test based on scenario
url, _ := mock.Upload(ctx, path, data, "application/pdf")
expectedURL := scenario.ExpectedOutput["public_url"].(string)
assert.Equal(t, expectedURL, url)
```

## 🎯 Quick Examples

### Example 1: Service Testing

```go
type ReportService struct {
    storage *mocks.SupabaseJSONMock
}

func TestReportService_Generate(t *testing.T) {
    service := &ReportService{
        storage: mocks.NewSupabaseJSONMockDefault(),
    }

    report := generateReport()
    url, err := service.storage.Upload(ctx, "reports/test.pdf", report, "application/pdf")

    require.NoError(t, err)
    assert.NotEmpty(t, url)
}
```

### Example 2: Validation Testing

```go
func TestFileValidation(t *testing.T) {
    mock := mocks.NewSupabaseJSONMockDefault()

    // Should fail - invalid content type
    _, err := mock.Upload(ctx, "test.exe", data, "application/x-msdownload")
    assert.Error(t, err)

    // Should fail - file too large
    largeData := make([]byte, mock.MaxFileSize + 1)
    _, err = mock.Upload(ctx, "big.pdf", largeData, "application/pdf")
    assert.Error(t, err)
}
```

### Example 3: Concurrent Testing

```go
func TestConcurrentUploads(t *testing.T) {
    mock := mocks.NewSupabaseJSONMockDefault()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            path := fmt.Sprintf("file-%d.pdf", idx)
            mock.Upload(ctx, path, []byte("content"), "application/pdf")
        }(i)
    }
    wg.Wait()

    assert.Equal(t, 100, mock.FileCount())
}
```

## 🔧 Configuration

### Default Configuration

```go
mock := mocks.NewSupabaseJSONMockDefault()

// Uses these defaults:
// - Project URL: https://mock-project.supabase.co
// - Bucket: reports
// - Max file size: 50MB (52428800 bytes)
// - Allowed types: application/pdf, application/json
```

### Custom JSON Configuration

Create `supabase_mock_data.json`:

```json
{
  "config": {
    "project_url": "https://your-project.supabase.co",
    "default_bucket": "your-bucket",
    "max_file_size": 10485760,
    "allowed_content_types": ["application/pdf"]
  },
  "storage": {
    "buckets": {
      "your-bucket": {
        "public": true,
        "files": {}
      }
    }
  }
}
```

Then load it:

```go
mock, err := mocks.NewSupabaseJSONMock("supabase_mock_data.json")
```

## 🐛 Debugging Tips

### Export Mock State

```go
// Upload some files
mock.Upload(ctx, "test1.pdf", data1, "application/pdf")
mock.Upload(ctx, "test2.pdf", data2, "application/pdf")

// Export current state
jsonData, _ := mock.ExportToJSON()
fmt.Println(string(jsonData))
```

### List All Files

```go
paths := mock.ListFiles()
for _, path := range paths {
    fmt.Printf("Stored file: %s\n", path)
}
```

### Check File Count

```go
fmt.Printf("Files in storage: %d\n", mock.FileCount())
```

## 🧪 Test Patterns

### Setup/Teardown

```go
func TestWithSetup(t *testing.T) {
    // Setup
    mock := mocks.NewSupabaseJSONMockDefault()
    defer mock.Clear() // Cleanup

    // Test
    // ... your test code ...
}
```

### Table-Driven Tests

```go
func TestTableDriven(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        content []byte
        wantErr bool
    }{
        {"valid pdf", "test.pdf", []byte("pdf"), false},
        {"valid json", "test.json", []byte("{}"), false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := mocks.NewSupabaseJSONMockDefault()
            defer mock.Clear()

            _, err := mock.Upload(ctx, tt.path, tt.content, "application/pdf")
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## 📚 Next Steps

- **Full documentation**: See [SUPABASE_MOCK_USAGE.md](./SUPABASE_MOCK_USAGE.md)
- **Example tests**: See [example_usage_test.go](./example_usage_test.go)
- **Unit tests**: See [supabase_json_mock_test.go](./supabase_json_mock_test.go)

## ⚡ Performance

- ✅ In-memory (no disk I/O)
- ✅ Thread-safe (safe for concurrent use)
- ✅ No network calls (instant operations)
- ✅ Minimal overhead (perfect for test suites)

## 🔐 Allowed Content Types

By default, these content types are allowed:

- ✅ `application/pdf`
- ✅ `application/json`
- ✅ `text/plain` (when loaded from JSON)

To allow other types, modify `supabase_mock_data.json`:

```json
{
  "config": {
    "allowed_content_types": [
      "application/pdf",
      "application/json",
      "image/png",
      "image/jpeg"
    ]
  }
}
```

## 💡 Pro Tips

1. **Always defer Clear()** to prevent test pollution
2. **Use JSON scenarios** for complex test cases
3. **Test concurrent access** if your code uses goroutines
4. **Export state** when debugging failures
5. **Check errors** - mock validates content types and sizes

---

**Ready to test?** Copy the "Write Your First Test" example and start coding! 🚀
