package remote

import (
	"os"
	"strings"
	"testing"
)

func TestFileUploaderConfig(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:   "http://127.0.0.1:8080",
		FilePath:  "./test.txt",
		ChunkSize: 1024 * 1024,
	}

	if config.BaseURL != "http://127.0.0.1:8080" {
		t.Errorf("Expected BaseURL 'http://127.0.0.1:8080', got '%s'", config.BaseURL)
	}

	if config.FilePath != "./test.txt" {
		t.Errorf("Expected FilePath './test.txt', got '%s'", config.FilePath)
	}

	if config.ChunkSize != 1024*1024 {
		t.Errorf("Expected ChunkSize %d, got %d", 1024*1024, config.ChunkSize)
	}
}

func TestNewFileUploader(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "./test.txt",
	}

	uploader := NewFileUploader(config)

	if uploader == nil {
		t.Fatal("NewFileUploader should not return nil")
	}

	if uploader.config.BaseURL != "http://127.0.0.1:8080" {
		t.Errorf("Expected BaseURL 'http://127.0.0.1:8080', got '%s'", uploader.config.BaseURL)
	}

	if uploader.config.ChunkSize != DEFAULT_CHUNK_SIZE {
		t.Errorf("Expected default ChunkSize %d, got %d", DEFAULT_CHUNK_SIZE, uploader.config.ChunkSize)
	}
}

func TestNewFileUploader_WithCustomChunkSize(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:   "http://127.0.0.1:8080",
		FilePath:  "./test.txt",
		ChunkSize: 8 * 1024 * 1024,
	}

	uploader := NewFileUploader(config)

	if uploader.config.ChunkSize != 8*1024*1024 {
		t.Errorf("Expected ChunkSize %d, got %d", 8*1024*1024, uploader.config.ChunkSize)
	}
}

func TestFileUploader_getFileMD5(t *testing.T) {
	tempDir := t.TempDir()
	testFile := tempDir + "/test.txt"
	testContent := "Test content for MD5 calculation"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: testFile,
	}

	uploader := NewFileUploader(config)
	md5, err := uploader.getFileMD5(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate MD5: %v", err)
	}

	if md5 == "" {
		t.Error("MD5 should not be empty")
	}

	if len(md5) != 32 {
		t.Errorf("MD5 should be 32 characters, got %d", len(md5))
	}
}

func TestFileUploader_getFileMD5_NonExistent(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "/nonexistent/file.txt",
	}

	uploader := NewFileUploader(config)
	_, err := uploader.getFileMD5("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when calculating MD5 for non-existent file")
	}
}

func TestFileUploader_checkUploadedSize(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "./test.txt",
	}

	uploader := NewFileUploader(config)
	_, err := uploader.checkUploadedSize("test_md5", "test.txt")
	if err == nil {
		t.Error("Expected error when checking uploaded size without server")
	}

	if err != nil && !strings.Contains(err.Error(), "查询失败") {
		t.Errorf("Expected query error, got: %v", err)
	}
}

func TestFileUploader_uploadChunk(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "./test.txt",
	}

	uploader := NewFileUploader(config)
	chunk := []byte("test chunk data")
	_, err := uploader.uploadChunk("test_md5", "test.txt", 0, chunk)
	if err == nil {
		t.Error("Expected error when uploading chunk without server")
	}

	if err != nil && !strings.Contains(err.Error(), "分片上传失败") {
		t.Errorf("Expected upload error, got: %v", err)
	}
}

func TestFileUploader_Upload_NonExistent(t *testing.T) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "/nonexistent/file.txt",
	}

	uploader := NewFileUploader(config)
	err := uploader.Upload()
	if err == nil {
		t.Error("Expected error when uploading non-existent file")
	}

	if err != nil && !strings.Contains(err.Error(), "文件不存在") {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

func TestFileUploader_Upload(t *testing.T) {
	// tempDir := t.TempDir()
	// testFile := tempDir + "/test.txt"
	testFile := "./test.txt"
	//testContent := strings.Repeat("test content ", 100)

	// err := os.WriteFile(testFile, []byte(testContent), 0644)
	// if err != nil {
	// 	t.Fatalf("Failed to create test file: %v", err)
	// }

	config := FileUploaderConfig{
		BaseURL:        "http://127.0.0.1:8080",
		FilePath:       testFile,
		RemoteFileName: "task1_step1-test_large_file.zip",
		ChunkSize:      1 * 1024,
	}

	uploader := NewFileUploader(config)
	err := uploader.Upload()
	if err != nil {
		t.Error("Expected error when uploading without server")
	}

	if err != nil && !strings.Contains(err.Error(), "查询已上传大小失败") {
		t.Errorf("Expected check size error, got: %v", err)
	}
}

func TestFileUploader_Integration(t *testing.T) {
	t.Skip("跳过集成测试，需要真实的上传服务器")

	tempDir := t.TempDir()
	testFile := tempDir + "/test_large_file.zip"
	testFileContent := strings.Repeat("test data for upload ", 10000)

	err := os.WriteFile(testFile, []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := FileUploaderConfig{
		BaseURL:   "http://127.0.0.1:8080",
		FilePath:  testFile,
		ChunkSize: 4 * 1024 * 1024,
	}

	uploader := NewFileUploader(config)
	err = uploader.Upload()
	if err != nil {
		t.Fatalf("上传失败：%v", err)
	}
}

func BenchmarkFileUploader_NewFileUploader(b *testing.B) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "./test.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewFileUploader(config)
	}
}

func BenchmarkFileUploader_getFileMD5(b *testing.B) {
	tempDir := b.TempDir()
	testFile := tempDir + "/test.txt"
	testContent := strings.Repeat("test content ", 1000)

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: testFile,
	}

	uploader := NewFileUploader(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uploader.getFileMD5(testFile)
	}
}

func BenchmarkFileUploader_uploadChunk(b *testing.B) {
	config := FileUploaderConfig{
		BaseURL:  "http://127.0.0.1:8080",
		FilePath: "./test.txt",
	}

	uploader := NewFileUploader(config)
	chunk := make([]byte, 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uploader.uploadChunk("test_md5", "test.txt", int64(i)*1024*1024, chunk)
	}
}
