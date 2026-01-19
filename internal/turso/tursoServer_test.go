package turso

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

// --- Mock Server ---

// MockTursoServer is a configurable mock HTTP server for testing uploads
type MockTursoServer struct {
	*httptest.Server

	mu              sync.Mutex
	receivedHeaders map[string][]string
	requestCount    int
	chunkData       map[int][]byte
	uploadID        string

	// Configurable responses
	startUploadStatus int
	chunkUploadStatus int
	finalizeStatus    int
	chunkSize         int64

	// Error simulation
	failAtChunk       int // -1 means no failure
	failAtChunkStatus int // Status code to return when failing (default 400 - non-retriable)
	failAtEndpoint    string
}

func NewMockTursoServer() *MockTursoServer {
	mock := &MockTursoServer{
		chunkData:         make(map[int][]byte),
		receivedHeaders:   make(map[string][]string),
		startUploadStatus: http.StatusOK,
		chunkUploadStatus: http.StatusOK,
		finalizeStatus:    http.StatusOK,
		chunkSize:         1024 * 1024, // 1MB default
		failAtChunk:       -1,
		failAtChunkStatus: http.StatusTeapot, // Non-retriable, to avoid holding up the test
		uploadID:          "test-upload-id",
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		mock.requestCount++

		// Store headers
		for k, v := range r.Header {
			mock.receivedHeaders[k] = v
		}

		switch {
		case r.Method == "PUT" && r.URL.Path == "/v2/upload/start":
			mock.handleMultipartStart(w, r)
		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/v2/upload/"+mock.uploadID+"/chunk/"):
			mock.handleChunkUpload(w, r)
		case r.Method == "PUT" && r.URL.Path == "/v2/upload/"+mock.uploadID+"/finalize":
			mock.handleFinalize(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return mock
}

func (m *MockTursoServer) handleMultipartStart(w http.ResponseWriter, r *http.Request) {
	if m.failAtEndpoint == "start" {
		w.WriteHeader(m.startUploadStatus)
		w.Write([]byte(`{"error": "simulated start error"}`))
		return
	}

	w.WriteHeader(m.startUploadStatus)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chunk_size": m.chunkSize,
		"upload_id":  m.uploadID,
	})
}

func (m *MockTursoServer) handleChunkUpload(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	chunkID, _ := strconv.Atoi(parts[len(parts)-1])

	if m.failAtChunk == chunkID {
		w.WriteHeader(m.failAtChunkStatus)
		w.Write([]byte(`{"error": "simulated chunk error"}`))
		return
	}

	data, _ := io.ReadAll(r.Body)
	m.chunkData[chunkID] = data
	w.WriteHeader(m.chunkUploadStatus)
}

func (m *MockTursoServer) handleFinalize(w http.ResponseWriter, r *http.Request) {
	if m.failAtEndpoint == "finalize" {
		w.WriteHeader(m.finalizeStatus)
		w.Write([]byte(`{"error": "simulated finalize error"}`))
		return
	}

	w.WriteHeader(m.finalizeStatus)
}

// GetAllChunkData reconstructs the full data from all chunks in order
func (m *MockTursoServer) GetAllChunkData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []byte
	for i := 0; i < len(m.chunkData); i++ {
		result = append(result, m.chunkData[i]...)
	}
	return result
}

// GetHeader returns the first value for a header key
func (m *MockTursoServer) GetHeader(key string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if values, ok := m.receivedHeaders[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

// --- Test Helpers ---

// createTestFileWithContent creates a temp file with specific content
func createTestFileWithContent(t *testing.T, content []byte) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-db-*.sqlite")
	require.NoError(t, err)

	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	t.Cleanup(func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	})

	return tmpFile.Name()
}

// createTestFile creates a temp file with deterministic data of specified size
func createTestFile(t *testing.T, size int64) string {
	t.Helper()

	pattern := []byte("TESTDATA")
	content := make([]byte, size)
	for i := int64(0); i < size; i++ {
		content[i] = pattern[i%int64(len(pattern))]
	}

	return createTestFileWithContent(t, content)
}

// createTestClient creates a TursoServerClient pointing to the mock server
func createTestClient(t *testing.T, serverURL string) *TursoServerClient {
	t.Helper()

	baseURL, err := url.Parse(serverURL)
	require.NoError(t, err)

	tokenProvider := func() (string, error) {
		return "test-token", nil
	}

	client, err := NewTursoServerClient(baseURL, tokenProvider, 5*time.Minute, "test-version", "test-org")
	require.NoError(t, err)

	return &client
}

// --- Progress Recorder ---

// ProgressCall records a single progress callback invocation
type ProgressCall struct {
	ProgressPct   int
	UploadedBytes int64
	TotalBytes    int64
	ElapsedTime   time.Duration
	Done          bool
}

// ProgressRecorder tracks all progress callback invocations
type ProgressRecorder struct {
	mu    sync.Mutex
	calls []ProgressCall
}

func NewProgressRecorder() *ProgressRecorder {
	return &ProgressRecorder{}
}

func (pr *ProgressRecorder) Callback() func(int, int64, int64, time.Duration, bool) {
	return func(progressPct int, uploadedBytes, totalBytes int64, elapsedTime time.Duration, done bool) {
		pr.mu.Lock()
		defer pr.mu.Unlock()
		pr.calls = append(pr.calls, ProgressCall{
			ProgressPct:   progressPct,
			UploadedBytes: uploadedBytes,
			TotalBytes:    totalBytes,
			ElapsedTime:   elapsedTime,
			Done:          done,
		})
	}
}

func (pr *ProgressRecorder) GetCalls() []ProgressCall {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return append([]ProgressCall{}, pr.calls...)
}

func (pr *ProgressRecorder) CallCount() int {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return len(pr.calls)
}

func (pr *ProgressRecorder) WasDoneCalled() bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	for _, call := range pr.calls {
		if call.Done {
			return true
		}
	}
	return false
}

// VerifyProgressIncreasing checks that progress never decreases
func (pr *ProgressRecorder) VerifyProgressIncreasing(t *testing.T) {
	t.Helper()
	calls := pr.GetCalls()

	lastPct := -1
	for i, call := range calls {
		require.GreaterOrEqual(t, call.ProgressPct, lastPct, "Progress decreased at call %d: %d -> %d", i, lastPct, call.ProgressPct)
		lastPct = call.ProgressPct
	}
}

// VerifyFinalCall checks that the final callback has correct values
func (pr *ProgressRecorder) VerifyFinalCall(t *testing.T, expectedTotal int64) {
	t.Helper()
	calls := pr.GetCalls()

	require.NotEmpty(t, calls, "No progress callbacks received")

	lastCall := calls[len(calls)-1]
	require.True(t, lastCall.Done, "Last call did not have Done=true")
	require.Equal(t, 100, lastCall.ProgressPct, "Last call progress was not 100%%")
	require.Equal(t, expectedTotal, lastCall.TotalBytes, "Last call total bytes mismatch")
}

// --- Multipart Upload Tests ---

func TestUploadFileMultipart_Success(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024 // 1KB chunks
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 5*1024) // 5KB = 5 chunks
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 5, numChunks)

	allData := mock.GetAllChunkData()
	require.Len(t, allData, 5*1024)

	progress.VerifyFinalCall(t, 5*1024)
}

func TestUploadFileMultipart_SingleChunk(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 10 * 1024 // 10KB chunks
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 1024) // 1KB file = 1 chunk
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 1, numChunks)
}

func TestUploadFileMultipart_ExactlyDivisible(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024 // 1KB chunks
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 3*1024) // 3KB = exactly 3 chunks
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 3, numChunks)

	// Verify each chunk is exactly 1024 bytes
	for i := 0; i < 3; i++ {
		mock.mu.Lock()
		chunkLen := len(mock.chunkData[i])
		mock.mu.Unlock()

		require.Equal(t, 1024, chunkLen, "Chunk %d size mismatch", i)
	}
}

func TestUploadFileMultipart_NotDivisible(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024 // 1KB chunks
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 2500) // 2500 bytes = 2 full + 1 partial (452 bytes)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	chunk0Len := len(mock.chunkData[0])
	chunk1Len := len(mock.chunkData[1])
	chunk2Len := len(mock.chunkData[2])
	mock.mu.Unlock()

	require.Equal(t, 3, numChunks)
	require.Equal(t, 1024, chunk0Len, "Chunk 0 size mismatch")
	require.Equal(t, 1024, chunk1Len, "Chunk 1 size mismatch")
	require.Equal(t, 452, chunk2Len, "Chunk 2 size mismatch (remainder)") // 2500 - 1024 - 1024 = 452
}

func TestUploadFileMultipart_WithEncryption(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 2048)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "aes-256-cbc", "base64key==", progress.Callback())
	require.NoError(t, err)

	require.Equal(t, "aes-256-cbc", mock.GetHeader("X-Turso-Encryption-Cipher"))
	require.Equal(t, "base64key==", mock.GetHeader("X-Turso-Encryption-Key"))
}

func TestUploadFileMultipart_StartFailure(t *testing.T) {
	mock := NewMockTursoServer()
	mock.startUploadStatus = http.StatusInternalServerError
	mock.failAtEndpoint = "start"
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 2048)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "initiate multipart upload failed")
}

func TestUploadFileMultipart_ChunkFailure_FirstChunk(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	mock.failAtChunk = 0 // Fail on first chunk
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 3*1024)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "chunk 0")
}

func TestUploadFileMultipart_ChunkFailure_MiddleChunk(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	mock.failAtChunk = 2 // Fail on third chunk
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 5*1024) // 5 chunks
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "chunk 2")

	// Verify first 2 chunks were uploaded
	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 2, numChunks, "Expected 2 successful chunks before failure")
}

func TestUploadFileMultipart_ChunkFailure_LastChunk(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	mock.failAtChunk = 4 // Fail on fifth (last) chunk
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 5*1024) // 5 chunks
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "chunk 4")
}

func TestUploadFileMultipart_FinalizeFailure(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	mock.finalizeStatus = http.StatusInternalServerError
	mock.failAtEndpoint = "finalize"
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 2*1024)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "finalize multipart upload failed")

	// Verify all chunks were uploaded before finalize failed
	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 2, numChunks, "Expected 2 chunks uploaded before finalize")
}

func TestUploadFileMultipart_FileNotFound(t *testing.T) {
	mock := NewMockTursoServer()
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart("/nonexistent/path/file.db", "", "", progress.Callback())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open file")
}

func TestUploadFileMultipart_ProgressCallbackPerChunk(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 1024
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 5*1024) // 5 chunks
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	calls := progress.GetCalls()

	// Should have at least 5 progress calls (one per chunk) plus final
	require.GreaterOrEqual(t, len(calls), 5, "Expected at least 5 progress callbacks")

	// Verify progress values match chunk boundaries (20%, 40%, 60%, 80%, 100%)
	expectedProgress := []int{20, 40, 60, 80, 100}
	for i, expected := range expectedProgress {
		if i < len(calls) {
			require.Equal(t, expected, calls[i].ProgressPct, "Call %d progress mismatch", i)
		}
	}

	// Final call should have Done=true
	require.True(t, calls[len(calls)-1].Done, "Final callback should have Done=true")
}

func TestUploadFileMultipart_ContentLengthHeader(t *testing.T) {
	var receivedContentLengths []string
	var mu sync.Mutex
	uploadID := "test-upload-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if strings.Contains(r.URL.Path, "/chunk/") {
			receivedContentLengths = append(receivedContentLengths, r.Header.Get("Content-Length"))
		}
		mu.Unlock()

		switch {
		case r.URL.Path == "/v2/upload/start":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"chunk_size": int64(1024), "upload_id": uploadID})
		case strings.HasPrefix(r.URL.Path, "/v2/upload/"+uploadID+"/chunk/"):
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/v2/upload/"+uploadID+"/finalize":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := createTestClient(t, server.URL)
	testFile := createTestFile(t, 2500) // 2 full chunks + 452 bytes
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	expectedLengths := []string{"1024", "1024", "452"}
	require.Len(t, receivedContentLengths, len(expectedLengths))

	for i, expected := range expectedLengths {
		require.Equal(t, expected, receivedContentLengths[i], "Chunk %d Content-Length mismatch", i)
	}
}

func TestUploadFileMultipart_DataIntegrity(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 5
	defer mock.Close()

	client := createTestClient(t, mock.URL)

	// Create file with known content
	knownData := []byte("This is known test data that will be verified after multipart upload to ensure integrity")
	testFile := createTestFileWithContent(t, knownData)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	// Reconstruct data from chunks
	receivedData := mock.GetAllChunkData()

	require.Equal(t, knownData, receivedData, "Data integrity check failed")
}

func TestUploadFileMultipart_ChunkSizeFromServer(t *testing.T) {
	// Test that client respects chunk size from server
	serverChunkSize := int64(512) // Very small chunks

	mock := NewMockTursoServer()
	mock.chunkSize = serverChunkSize
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 2000) // Should result in 4 chunks (512+512+512+464)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	mock.mu.Lock()
	numChunks := len(mock.chunkData)
	mock.mu.Unlock()

	require.Equal(t, 4, numChunks, "Expected 4 chunks with 512-byte chunk size")
}

func TestUploadFileMultipart_HTTPStatusCodes(t *testing.T) {
	// Note: Only testing non-retriable status codes here.
	// 5xx codes trigger retry logic which requires time mocking to test properly.
	testCases := []struct {
		name          string
		status        int
		shouldSucceed bool
	}{
		{"200 OK", http.StatusOK, true},
		{"201 Created", http.StatusCreated, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"403 Forbidden", http.StatusForbidden, false},
		{"404 Not Found", http.StatusNotFound, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockTursoServer()
			mock.chunkSize = 1024
			if !tc.shouldSucceed {
				mock.failAtChunkStatus = tc.status
				mock.failAtChunk = 0 // Fail on first chunk
			}
			defer mock.Close()

			client := createTestClient(t, mock.URL)
			testFile := createTestFile(t, 2048)
			progress := NewProgressRecorder()

			err := client.UploadFileMultipart(testFile, "", "", progress.Callback())

			if tc.shouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// --- Progress Reader Tests ---

func TestProgressReader_ProgressUpdates(t *testing.T) {
	data := make([]byte, 1000)
	reader := bytes.NewReader(data)

	var progressCalls []int
	var mu sync.Mutex

	now := time.Now()
	pr := &progressReader{
		reader:          reader,
		totalSize:       1000,
		lastUpdate:      -1,
		lastUpdateTime:  now,
		lastUpdateBytes: 0,
		onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
			mu.Lock()
			progressCalls = append(progressCalls, pct)
			mu.Unlock()
		},
		startTime: now,
	}

	buf := make([]byte, 100) // Read in 100-byte chunks
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	mu.Lock()
	defer mu.Unlock()

	require.NotEmpty(t, progressCalls, "Expected progress callbacks")

	// Verify progress increased
	var lastPct int
	for i, pct := range progressCalls {
		require.GreaterOrEqual(t, pct, lastPct, "Progress decreased at index %d: %d -> %d", i, lastPct, pct)
		lastPct = pct
	}

	require.Equal(t, 100, progressCalls[len(progressCalls)-1], "Final progress should be 100%%")
}

func TestProgressReader_DoneOnEOF(t *testing.T) {
	// The progressReader sets done=true when EOF is returned along with n > 0
	// in the same Read() call. This uses a custom reader that returns EOF with data.
	data := []byte("test data exactly")

	// Custom reader that returns data and EOF together
	reader := &testEOFReader{data: data}

	var lastDone bool
	var lastPct int
	now := time.Now()
	pr := &progressReader{
		reader:          reader,
		totalSize:       int64(len(data)),
		lastUpdate:      -1,
		lastUpdateTime:  now,
		lastUpdateBytes: 0,
		onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
			lastDone = done
			lastPct = pct
		},
		startTime: now,
	}

	// Read all at once
	buf := make([]byte, 200)
	n, err := pr.Read(buf)

	require.Equal(t, len(data), n)
	require.ErrorIs(t, err, io.EOF)
	require.True(t, lastDone, "Expected Done=true on final callback when EOF returned with data")
	require.Equal(t, 100, lastPct)
}

// testEOFReader is a custom reader that returns data and EOF in the same Read() call
type testEOFReader struct {
	data []byte
	read bool
}

func (r *testEOFReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}
	r.read = true
	n := copy(p, r.data)
	return n, io.EOF
}

func TestProgressReader_BytesReadAccurate(t *testing.T) {
	data := make([]byte, 500)
	reader := bytes.NewReader(data)

	var lastUploadedBytes int64
	now := time.Now()
	pr := &progressReader{
		reader:          reader,
		totalSize:       500,
		lastUpdate:      -1,
		lastUpdateTime:  now,
		lastUpdateBytes: 0,
		onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
			lastUploadedBytes = uploaded
		},
		startTime: now,
	}

	_, err := io.ReadAll(pr)
	require.NoError(t, err)

	require.Equal(t, int64(500), lastUploadedBytes)
}

func TestProgressReader_WithBaseBytes(t *testing.T) {
	// Simulate reading from a second chunk of a 200-byte total upload
	// where the first chunk (100 bytes) has already been uploaded.
	data := strings.Repeat("x", 100) // Second chunk: 100 bytes
	reader := strings.NewReader(data)

	var progressUpdates []struct {
		pct      int
		uploaded int64
		total    int64
	}

	now := time.Now()
	pr := &progressReader{
		reader:          reader,
		totalSize:       200,
		baseBytes:       100,
		startTime:       now,
		lastUpdateTime:  now,
		lastUpdateBytes: 100,
		onProgress: func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool) {
			progressUpdates = append(progressUpdates, struct {
				pct      int
				uploaded int64
				total    int64
			}{progressPct, uploadedBytes, totalBytes})
		},
		lastUpdate: 50,
	}

	buf := make([]byte, 10)
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	require.NotEmpty(t, progressUpdates, "expected progress updates")

	firstUpdate := progressUpdates[0]
	require.Greater(t, firstUpdate.pct, 50, "first progress update should be > 50%%")

	for i, update := range progressUpdates {
		require.Greater(t, update.uploaded, int64(100), "update %d: uploadedBytes should be > 100 (baseBytes)", i)
		require.Equal(t, int64(200), update.total, "update %d: totalBytes should be 200", i)
	}

	lastUpdate := progressUpdates[len(progressUpdates)-1]
	require.Equal(t, 100, lastUpdate.pct, "final progress should be 100%%")
	require.Equal(t, int64(200), lastUpdate.uploaded, "final uploadedBytes should be 200")
}

func TestProgressReader_CumulativeAcrossChunks(t *testing.T) {
	totalSize := int64(300)
	chunkSize := int64(100)

	var allUpdates []struct {
		pct      int
		uploaded int64
	}

	// Simulate reading 3 chunks
	var baseBytes int64 = 0
	lastPct := -1
	lastUpdateTime := time.Now()
	var lastUpdateBytes int64 = 0

	for chunk := 0; chunk < 3; chunk++ {
		data := bytes.Repeat([]byte("x"), int(chunkSize))
		reader := bytes.NewReader(data)

		pr := &progressReader{
			reader:          reader,
			totalSize:       totalSize,
			baseBytes:       baseBytes,
			startTime:       time.Now(),
			lastUpdateTime:  lastUpdateTime,
			lastUpdateBytes: lastUpdateBytes,
			onProgress: func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool) {
				allUpdates = append(allUpdates, struct {
					pct      int
					uploaded int64
				}{progressPct, uploadedBytes})
			},
			lastUpdate: lastPct,
		}

		// Read all of this chunk
		buf := make([]byte, 10)
		for {
			_, err := pr.Read(buf)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		baseBytes += chunkSize
		lastPct = pr.lastUpdate
		lastUpdateTime = pr.lastUpdateTime
		lastUpdateBytes = pr.lastUpdateBytes
	}

	require.NotEmpty(t, allUpdates, "expected progress updates")
	require.LessOrEqual(t, allUpdates[0].pct, 10, "first update should be low")
	require.Equal(t, 100, allUpdates[len(allUpdates)-1].pct, "last update should be 100%%")
	for i := 1; i < len(allUpdates); i++ {
		require.GreaterOrEqual(t, allUpdates[i].pct, allUpdates[i-1].pct, "progress went backwards: %d%% -> %d%%", allUpdates[i-1].pct, allUpdates[i].pct)
	}
	for i := 1; i < len(allUpdates); i++ {
		require.GreaterOrEqual(t, allUpdates[i].uploaded, allUpdates[i-1].uploaded, "uploadedBytes went backwards: %d -> %d", allUpdates[i-1].uploaded, allUpdates[i].uploaded)
	}
}

func TestUploadFileMultipart_SmoothProgress(t *testing.T) {
	mock := NewMockTursoServer()
	mock.chunkSize = 100 * 1024
	defer mock.Close()

	client := createTestClient(t, mock.URL)
	testFile := createTestFile(t, 1024*1024)
	progress := NewProgressRecorder()

	err := client.UploadFileMultipart(testFile, "", "", progress.Callback())
	require.NoError(t, err)

	calls := progress.GetCalls()

	require.Greater(t, len(calls), 15, "Expected smooth progress with many updates, got only %d", len(calls))
	progress.VerifyProgressIncreasing(t)
	require.True(t, calls[len(calls)-1].Done, "Final callback should have Done=true")
}

// infiniteReader provides unlimited bytes for large file simulation (never returns EOF)
type infiniteReader struct {
	bytesRead int64
}

func (r *infiniteReader) Read(p []byte) (int, error) {
	n := len(p)
	r.bytesRead += int64(n)
	return n, nil
}

func TestProgressReader_TimeBasedUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(100 * 1024 * 1024)

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          &infiniteReader{},
			totalSize:       totalSize,
			lastUpdate:      0,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Read 0.5% of data (not enough for 1% threshold)
		buf := make([]byte, 512*1024) // 0.5MB
		pr.Read(buf)
		initialCalls := len(calls)

		// Advance fake time by 3 seconds
		time.Sleep(3 * time.Second)
		synctest.Wait()

		// Read again - should trigger time-based update
		pr.Read(buf)

		require.Greater(t, len(calls), initialCalls, "Expected time-based callback after 2s")
	})
}

func TestProgressReader_ByteBasedUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(10 * 1024 * 1024 * 1024) // 10GB

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          &infiniteReader{},
			totalSize:       totalSize,
			lastUpdate:      0,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Read 60MB (more than 50MB threshold but less than 1% of 10GB)
		buf := make([]byte, 60*1024*1024)
		pr.Read(buf)

		require.NotEmpty(t, calls, "Expected byte-based callback after 50MB")
	})
}

func TestProgressReader_NoUnnecessaryUpdates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(100 * 1024 * 1024) // 100MB

		var callCount int
		now := time.Now()
		pr := &progressReader{
			reader:          &infiniteReader{},
			totalSize:       totalSize,
			lastUpdate:      50, // Already at 50%
			lastUpdateTime:  now,
			lastUpdateBytes: 50 * 1024 * 1024, // 50MB already uploaded
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				callCount++
			},
		}

		// Small read that doesn't cross any threshold
		// 0.5MB is 0.5% of 100MB - not enough for 1% progress
		// Not enough time passed, not enough bytes (< 50MB since last update)
		buf := make([]byte, 512*1024) // 0.5MB
		pr.Read(buf)

		require.Equal(t, 0, callCount, "Should not fire callback when no threshold crossed")
	})
}

func TestProgressReader_PercentageThreshold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(1000)
		data := make([]byte, totalSize)
		reader := bytes.NewReader(data)

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          reader,
			totalSize:       totalSize,
			lastUpdate:      -1,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Read 2% - should trigger (crosses 1% threshold)
		buf := make([]byte, 20) // 2%
		pr.Read(buf)

		require.Len(t, calls, 1)
		require.Equal(t, 2, calls[0].ProgressPct)
	})
}

func TestProgressReader_EOFAlwaysTriggers(t *testing.T) {
	// EOF should always trigger update, regardless of other thresholds
	synctest.Test(t, func(t *testing.T) {
		data := []byte("small")
		// Use testEOFReader which returns data AND EOF together in the same Read() call
		reader := &testEOFReader{data: data}

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          reader,
			totalSize:       int64(len(data)),
			lastUpdate:      100, // Set to 100 so percentage doesn't trigger (100 > 100 is false)
			lastUpdateTime:  now,
			lastUpdateBytes: int64(len(data)),
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Read to EOF - testEOFReader returns data and EOF together
		buf := make([]byte, 100)
		pr.Read(buf)

		require.NotEmpty(t, calls, "EOF should trigger callback even when other thresholds not met")
		require.True(t, calls[len(calls)-1].Done, "EOF should trigger done=true")
	})
}

func TestProgressReader_MultipleThresholdsCombined(t *testing.T) {
	// Test that any threshold being met triggers update
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(100 * 1024 * 1024 * 1024) // 100GB

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          &infiniteReader{},
			totalSize:       totalSize,
			lastUpdate:      0,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Scenario 1: Read 10MB (no threshold met - less than 50MB, less than 1% of 100GB)
		buf := make([]byte, 10*1024*1024)
		pr.Read(buf)
		require.Empty(t, calls, "Should not trigger with only 10MB read on 100GB file")

		// Scenario 2: Another 50MB should trigger byte threshold (total 60MB > 50MB)
		buf = make([]byte, 50*1024*1024)
		pr.Read(buf)
		require.Len(t, calls, 1, "Should trigger after 60MB total (>50MB byte threshold)")
	})
}

func TestProgressReader_TimeThresholdWithSmallReads(t *testing.T) {
	// Test time-based updates with small frequent reads (simulates slow network)
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(10 * 1024 * 1024 * 1024) // 10GB

		var calls []ProgressCall
		now := time.Now()
		pr := &progressReader{
			reader:          &infiniteReader{},
			totalSize:       totalSize,
			lastUpdate:      0,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress: func(pct int, uploaded, total int64, elapsed time.Duration, done bool) {
				calls = append(calls, ProgressCall{pct, uploaded, total, elapsed, done})
			},
		}

		// Small read - no trigger
		buf := make([]byte, 1024) // 1KB
		pr.Read(buf)
		require.Empty(t, calls)

		// Advance time by 2.5 seconds
		time.Sleep(2500 * time.Millisecond)
		synctest.Wait()

		// Another small read - should trigger due to time
		pr.Read(buf)
		require.Len(t, calls, 1, "Should trigger after 2s time threshold")
	})
}

func TestProgressReader_UpdatesTrackingFieldsCorrectly(t *testing.T) {
	// Verify lastUpdateTime and lastUpdateBytes are updated correctly
	synctest.Test(t, func(t *testing.T) {
		totalSize := int64(1000)
		data := make([]byte, totalSize)
		reader := bytes.NewReader(data)

		now := time.Now()
		pr := &progressReader{
			reader:          reader,
			totalSize:       totalSize,
			lastUpdate:      -1,
			lastUpdateTime:  now,
			lastUpdateBytes: 0,
			startTime:       now,
			onProgress:      func(int, int64, int64, time.Duration, bool) {},
		}

		// Advance fake time so time.Now() will return a different value when update happens
		time.Sleep(1 * time.Millisecond)
		synctest.Wait()

		initialTime := pr.lastUpdateTime

		// Trigger an update via percentage
		buf := make([]byte, 50) // 5%
		pr.Read(buf)

		require.True(t, pr.lastUpdateTime.After(initialTime), "lastUpdateTime should be updated to a later time")
		require.Equal(t, int64(50), pr.lastUpdateBytes, "lastUpdateBytes should track uploaded bytes")
	})
}
