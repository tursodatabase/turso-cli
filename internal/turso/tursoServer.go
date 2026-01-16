package turso

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	mathrand "math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	defaultMaxRetries    = 15
	baseRetryDelay       = 2 * time.Second
	maxRetryDelay        = 60 * time.Second
	retryJitterMaxMillis = 1000
)

// debugUpload returns true if TURSO_DEBUG_UPLOAD=1 is set.
// This provides upload-specific debug logging without the full HTTP dumps from --debug.
func debugUpload() bool {
	return os.Getenv("TURSO_DEBUG_UPLOAD") == "1"
}

// TokenProvider is a function that returns a fresh authentication token.
type TokenProvider func() (string, error)

// progressReader is a custom io.Reader that tracks progress of the upload
// and calls the onProgress callback with the progress of the upload.
type progressReader struct {
	reader          io.Reader
	totalSize       int64
	bytesRead       int64
	baseBytes       int64 // Bytes already uploaded before progressReader started
	startTime       time.Time
	onProgress      func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)
	lastUpdate      int       // Last reported progress percentage. Initially -1 to ensure first update is always sent.
	lastUpdateTime  time.Time // Last time a progress update was sent
	lastUpdateBytes int64     // Total bytes uploaded at last update
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.bytesRead += int64(n)
		totalUploaded := pr.baseBytes + pr.bytesRead
		progressPct := int(float64(totalUploaded) / float64(pr.totalSize) * 100)

		timeSinceLastUpdate := time.Since(pr.lastUpdateTime)
		bytesSinceLastUpdate := totalUploaded - pr.lastUpdateBytes

		// Update if: percentage changed OR 2s elapsed OR 50MB uploaded OR done
		shouldUpdate := progressPct > pr.lastUpdate ||
			timeSinceLastUpdate >= 2*time.Second ||
			bytesSinceLastUpdate >= 50*1024*1024 ||
			errors.Is(err, io.EOF)

		if shouldUpdate {
			elapsedTime := time.Since(pr.startTime)
			pr.lastUpdate = progressPct
			pr.lastUpdateTime = time.Now()
			pr.lastUpdateBytes = totalUploaded
			pr.onProgress(progressPct, totalUploaded, pr.totalSize, elapsedTime, errors.Is(err, io.EOF))
		}
	}
	return n, err
}

type TursoServerClient struct {
	tenant           string
	client           *Client
	tokenProvider    TokenProvider
	tokenTTL         time.Duration
	lastTokenRefresh time.Time
}

func NewTursoServerClient(baseURL *url.URL, tokenProvider TokenProvider, tokenTTL time.Duration, cliVersion string, org string) (TursoServerClient, error) {
	initialToken, err := tokenProvider()
	if err != nil {
		return TursoServerClient{}, fmt.Errorf("failed to get initial token: %w", err)
	}

	newClient := New(baseURL, initialToken, cliVersion, org)

	return TursoServerClient{
		tenant:           org,
		client:           newClient,
		tokenProvider:    tokenProvider,
		tokenTTL:         tokenTTL,
		lastTokenRefresh: time.Now(),
	}, nil
}

func (i *TursoServerClient) refreshTokenIfNeeded() error {
	if i.tokenProvider == nil {
		return nil
	}
	token, err := i.tokenProvider()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	i.client.SetToken(token)
	i.lastTokenRefresh = time.Now()
	return nil
}

// isRetriableError determines if an error should be retried.
// Returns true for network errors, server errors (5xx), and specific client errors (408, 429).
func isRetriableError(err error, statusCode int) bool {
	// Network errors are always retriable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Connection errors (no response received)
	if err != nil && statusCode == 0 {
		return true
	}

	// Server errors (5xx) are retriable
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// Specific client errors that are retriable
	if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests {
		return true
	}

	return false
}

// calculateBackoff returns the backoff duration for a given retry attempt.
// Uses exponential backoff with jitter.
func calculateBackoff(attempt int) time.Duration {
	delay := baseRetryDelay * time.Duration(1<<uint(attempt)) // 2^attempt
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	// Add jitter
	jitter := time.Duration(mathrand.IntN(retryJitterMaxMillis)) * time.Millisecond
	return delay + jitter
}

// chunkUploadContext holds the context needed for uploading a chunk with retry support.
type chunkUploadContext struct {
	chunkID          int
	chunkPath        string
	chunkSize        int64
	chunkStartOffset int64 // File offset where this chunk starts
	file             *os.File
	headers          map[string]string
	totalSize        int64
	startTime        time.Time
	onProgress       func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)
	lastProgressPct  int
	lastUpdateTime   time.Time
	lastUpdateBytes  int64
}

// chunkUploadResult holds the progress state after a successful chunk upload.
type chunkUploadResult struct {
	lastProgressPct int
	lastUpdateTime  time.Time
	lastUpdateBytes int64
}

// uploadChunkWithRetry uploads a single chunk with retry logic.
// It handles token refresh, exponential backoff, and progress tracking reset on retry.
func (i *TursoServerClient) uploadChunkWithRetry(ctx *chunkUploadContext, maxRetries int) (chunkUploadResult, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if debugUpload() && attempt > 0 {
			log.Printf("[upload] Chunk %d: retry attempt %d/%d after error: %v", ctx.chunkID, attempt, maxRetries, lastErr)
		}

		// Refresh token on retries (ensures token doesn't expire during retry sequence)
		if attempt > 0 {
			if debugUpload() {
				log.Printf("[upload] Chunk %d: refreshing token before retry attempt %d", ctx.chunkID, attempt)
			}
			if err := i.refreshTokenIfNeeded(); err != nil {
				return chunkUploadResult{}, err
			}
		}

		// Seek to chunk start position for this attempt
		if _, err := ctx.file.Seek(ctx.chunkStartOffset, io.SeekStart); err != nil {
			return chunkUploadResult{}, fmt.Errorf("failed to seek to chunk %d start: %w", ctx.chunkID, err)
		}

		// Create a fresh reader for this attempt
		chunkReader := io.LimitReader(ctx.file, ctx.chunkSize)

		progressTracker := &progressReader{
			reader:          chunkReader,
			totalSize:       ctx.totalSize,
			baseBytes:       ctx.chunkStartOffset,
			startTime:       ctx.startTime,
			onProgress:      ctx.onProgress,
			lastUpdate:      ctx.lastProgressPct,
			lastUpdateTime:  ctx.lastUpdateTime,
			lastUpdateBytes: ctx.lastUpdateBytes,
		}

		// Attempt the upload
		r, err := i.client.PutBinary(ctx.chunkPath, progressTracker, ctx.headers)

		// Determine status code (0 if no response)
		statusCode := 0
		if r != nil {
			statusCode = r.StatusCode
		}

		// Success case
		if err == nil && (statusCode == http.StatusOK || statusCode == http.StatusCreated) {
			_ = r.Body.Close()
			if debugUpload() && attempt > 0 {
				log.Printf("[upload] Chunk %d: succeeded on attempt %d", ctx.chunkID, attempt+1)
			}
			return chunkUploadResult{
				lastProgressPct: progressTracker.lastUpdate,
				lastUpdateTime:  progressTracker.lastUpdateTime,
				lastUpdateBytes: progressTracker.lastUpdateBytes,
			}, nil
		}

		// Build error for this attempt
		if err != nil {
			lastErr = fmt.Errorf("failed to upload chunk %d: %w", ctx.chunkID, err)
		} else {
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			lastErr = fmt.Errorf("upload chunk %d failed with status code %d: %s", ctx.chunkID, statusCode, string(body))
		}

		// Check if error is retriable
		if !isRetriableError(err, statusCode) {
			if debugUpload() {
				log.Printf("[upload] Chunk %d: non-retriable error (status=%d): %v", ctx.chunkID, statusCode, lastErr)
			}
			return chunkUploadResult{}, lastErr
		}

		// Don't sleep after the last attempt
		if attempt < maxRetries {
			backoff := calculateBackoff(attempt)
			if debugUpload() {
				log.Printf("[upload] Chunk %d: retriable error (status=%d), waiting %v before retry", ctx.chunkID, statusCode, backoff)
			}
			time.Sleep(backoff)
		}
	}

	if debugUpload() {
		log.Printf("[upload] Chunk %d: exhausted all %d retries, giving up", ctx.chunkID, maxRetries+1)
	}
	return chunkUploadResult{}, fmt.Errorf("failed after %d retries: %w", maxRetries+1, lastErr)
}

// UploadFileMultipart uploads a database file using the multipart upload flow.
func (i *TursoServerClient) UploadFileMultipart(filepath string, remoteEncryptionCipher, remoteEncryptionKey string, onUploadProgress func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	// locking is on a best effort basis
	if unlock, err := lockFileExclusive(file); err == nil {
		defer unlock()
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats for %s: %w", filepath, err)
	}

	totalSize := stat.Size()
	startTime := time.Now()

	uploadStart, err := i.startMultipartUpload(totalSize)
	if err != nil {
		return err
	}

	uploadedBytes, err := i.uploadChunks(uploadStart.UploadID, uploadStart.ChunkSize, file, totalSize, startTime, remoteEncryptionCipher, remoteEncryptionKey, onUploadProgress)
	if err != nil {
		return err
	}

	if err := i.refreshTokenIfNeeded(); err != nil {
		return err
	}

	if err = i.finalizeUpload(uploadStart.UploadID); err != nil {
		return err
	}

	elapsedTime := time.Since(startTime)
	onUploadProgress(100, uploadedBytes, totalSize, elapsedTime, true)

	return nil
}

type multipartUploadStart struct {
	ChunkSize int64
	UploadID  string
}

func (i *TursoServerClient) startMultipartUpload(dbSize int64) (multipartUploadStart, error) {
	requestBody := map[string]int64{
		"db_size_bytes": dbSize,
	}

	body, err := marshal(requestBody)
	if err != nil {
		return multipartUploadStart{}, fmt.Errorf("failed to marshal multipart upload request: %w", err)
	}

	r, err := i.client.Put("/v2/upload/start", body)
	if err != nil {
		return multipartUploadStart{}, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return multipartUploadStart{}, fmt.Errorf("initiate multipart upload failed with status code %d and error reading response: %v", r.StatusCode, err)
		}
		return multipartUploadStart{}, fmt.Errorf("initiate multipart upload failed with status code %d: %s", r.StatusCode, string(body))
	}

	type multipartUploadResponse struct {
		ChunkSize int64  `json:"chunk_size"`
		UploadID  string `json:"upload_id"`
	}
	var uploadResp multipartUploadResponse
	if err := json.NewDecoder(r.Body).Decode(&uploadResp); err != nil {
		return multipartUploadStart{}, fmt.Errorf("failed to decode multipart upload response: %w", err)
	}

	return multipartUploadStart(uploadResp), nil
}

func (i *TursoServerClient) uploadChunks(uploadID string, chunkSize int64, file *os.File, totalSize int64, startTime time.Time, remoteEncryptionCipher, remoteEncryptionKey string, onUploadProgress func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)) (int64, error) {
	var uploadedBytes int64 = 0
	chunkID := 0
	lastProgressPct := -1
	lastUpdateTime := time.Now()
	var lastUpdateBytes int64 = 0

	totalChunks := (totalSize + chunkSize - 1) / chunkSize
	if debugUpload() {
		log.Printf("[upload] Starting chunked upload: uploadID=%s, totalSize=%d bytes, chunkSize=%d bytes, totalChunks=%d", uploadID, totalSize, chunkSize, totalChunks)
	}

	for uploadedBytes < totalSize {
		// Refresh token between chunks (not before first chunk)
		if chunkID > 0 {
			if err := i.refreshTokenIfNeeded(); err != nil {
				return 0, err
			}
		}

		remaining := totalSize - uploadedBytes
		currentChunkSize := chunkSize
		if remaining < chunkSize {
			currentChunkSize = remaining
		}

		chunkPath := fmt.Sprintf("/v2/upload/%s/chunk/%d", uploadID, chunkID)

		if debugUpload() {
			log.Printf("[upload] Uploading chunk %d/%d: size=%d bytes, offset=%d", chunkID+1, totalChunks, currentChunkSize, uploadedBytes)
		}

		headers := map[string]string{}
		if remoteEncryptionCipher != "" && remoteEncryptionKey != "" {
			headers[EncryptionCipherHeader] = remoteEncryptionCipher
			headers[EncryptionKeyHeader] = remoteEncryptionKey
		}
		headers["Content-Length"] = strconv.FormatInt(currentChunkSize, 10)

		ctx := &chunkUploadContext{
			chunkID:          chunkID,
			chunkPath:        chunkPath,
			chunkSize:        currentChunkSize,
			chunkStartOffset: uploadedBytes,
			file:             file,
			headers:          headers,
			totalSize:        totalSize,
			startTime:        startTime,
			onProgress:       onUploadProgress,
			lastProgressPct:  lastProgressPct,
			lastUpdateTime:   lastUpdateTime,
			lastUpdateBytes:  lastUpdateBytes,
		}

		result, err := i.uploadChunkWithRetry(ctx, defaultMaxRetries)
		if err != nil {
			return 0, err
		}

		uploadedBytes += currentChunkSize
		lastProgressPct = result.lastProgressPct
		lastUpdateTime = result.lastUpdateTime
		lastUpdateBytes = result.lastUpdateBytes

		chunkID++
	}
	return uploadedBytes, nil
}

func (i *TursoServerClient) finalizeUpload(uploadID string) error {
	r, err := i.client.Put(fmt.Sprintf("/v2/upload/%s/finalize", uploadID), nil)
	if err != nil {
		return fmt.Errorf("failed to finalize multipart upload: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("finalize multipart upload failed with status code %d and error reading response: %v", r.StatusCode, err)
		}
		return fmt.Errorf("finalize multipart upload failed with status code %d: %s", r.StatusCode, string(body))
	}
	return nil
}

type ExportInfo struct {
	CurrentGeneration int `json:"current_generation"`
}

const EncryptionKeyHeader = "x-turso-encryption-key"
const EncryptionCipherHeader = "x-turso-encryption-cipher"

func (i *TursoServerClient) Export(outputFile string, withMetadata bool, remoteEncryptionKey string) error {
	headers := map[string]string{}
	if remoteEncryptionKey != "" {
		headers[EncryptionKeyHeader] = remoteEncryptionKey
	}
	res, err := i.client.GetWithHeaders("/info", nil, headers)
	if err != nil {
		return fmt.Errorf("failed to fetch database info: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	var info ExportInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to decode /info response: %w", err)
	}

	exportRes, err := i.client.GetWithHeaders(fmt.Sprintf("/export/%d", info.CurrentGeneration), nil, headers)
	if err != nil {
		return fmt.Errorf("failed to fetch export: %w", err)
	}
	defer exportRes.Body.Close()
	if exportRes.StatusCode != http.StatusOK {
		return parseResponseError(exportRes)
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, exportRes.Body); err != nil {
		return fmt.Errorf("failed to write export to file: %w", err)
	}

	lastFrameNo, err := i.ExportWAL(outputFile, &info, remoteEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to export WAL: %w", err)
	}
	if withMetadata {
		if err := i.ExportMetadata(outputFile, &info, lastFrameNo); err != nil {
			return fmt.Errorf("failed to export metadata: %w", err)
		}
	}

	return nil
}

func (i *TursoServerClient) ExportWAL(outputFile string, info *ExportInfo, remoteEncryptionKey string) (int, error) {
	walFile := outputFile + "-wal"
	walOut, err := os.Create(walFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create WAL file: %w", err)
	}
	defer walOut.Close()

	var saltBytes [8]byte
	if _, err := rand.Read(saltBytes[:]); err != nil {
		return 0, fmt.Errorf("failed to generate random salt values: %w", err)
	}
	salt1 := binary.BigEndian.Uint32(saltBytes[0:4]) // Random salt-1
	salt2 := binary.BigEndian.Uint32(saltBytes[4:8]) // Random salt-2

	walHeader := make([]byte, 32)
	binary.BigEndian.PutUint32(walHeader[0:4], 0x377f0682) // Magic number
	binary.BigEndian.PutUint32(walHeader[4:8], 3007000)    // File format version
	binary.BigEndian.PutUint32(walHeader[8:12], 4096)      // Database page size
	binary.BigEndian.PutUint32(walHeader[12:16], 0)        // Checkpoint sequence number
	binary.BigEndian.PutUint32(walHeader[16:20], salt1)    // Salt-1 (must match frames)
	binary.BigEndian.PutUint32(walHeader[20:24], salt2)    // Salt-2 (must match frames)

	s0 := uint32(0)
	s1 := uint32(0)

	for i := 0; i < 24; i += 8 {
		x0 := binary.LittleEndian.Uint32(walHeader[i : i+4])
		x1 := binary.LittleEndian.Uint32(walHeader[i+4 : i+8])
		s0 += x0 + s1
		s1 += x1 + s0
	}

	binary.BigEndian.PutUint32(walHeader[24:28], s0)
	binary.BigEndian.PutUint32(walHeader[28:32], s1)

	if _, err := walOut.Write(walHeader); err != nil {
		return 0, fmt.Errorf("failed to write WAL header: %w", err)
	}

	const batchSize = 128
	frameNo := 1
	lastFrameNo := 0
	headers := map[string]string{}
	if remoteEncryptionKey != "" {
		headers[EncryptionKeyHeader] = remoteEncryptionKey
	}

	for {
		walRes, err := i.client.GetWithHeaders(fmt.Sprintf("/sync/%d/%d/%d", info.CurrentGeneration, frameNo, frameNo+batchSize), nil, headers)
		if err != nil {
			if frameNo == 1 {
				break
			}
			return lastFrameNo, fmt.Errorf("failed to fetch WAL frames: %w", err)
		}

		if walRes.StatusCode == http.StatusBadRequest || walRes.StatusCode == http.StatusInternalServerError {
			walRes.Body.Close()
			break
		}
		if walRes.StatusCode != http.StatusOK {
			walRes.Body.Close()
			if frameNo == 1 {
				break
			}
			return lastFrameNo, parseResponseError(walRes)
		}

		frames, err := io.ReadAll(walRes.Body)
		walRes.Body.Close()
		if err != nil {
			return lastFrameNo, fmt.Errorf("failed to read WAL frames: %w", err)
		}

		if len(frames) == 0 {
			break
		}

		frameSize := 4120
		framesInBatch := len(frames) / frameSize

		for i := 0; i < framesInBatch; i++ {
			offset := i * frameSize
			if offset+frameSize > len(frames) {
				return lastFrameNo, fmt.Errorf("invalid frame data: expected %d bytes, got %d", frameSize, len(frames)-offset)
			}
			frame := frames[offset : offset+frameSize]

			binary.BigEndian.PutUint32(frame[8:12], salt1)
			binary.BigEndian.PutUint32(frame[12:16], salt2)

			x0 := binary.LittleEndian.Uint32(frame[0:4])
			x1 := binary.LittleEndian.Uint32(frame[4:8])
			s0 += x0 + s1
			s1 += x1 + s0

			for j := 24; j < frameSize; j += 8 {
				x0 := binary.LittleEndian.Uint32(frame[j : j+4])
				x1 := binary.LittleEndian.Uint32(frame[j+4 : j+8])
				s0 += x0 + s1
				s1 += x1 + s0
			}

			binary.BigEndian.PutUint32(frame[16:20], s0)
			binary.BigEndian.PutUint32(frame[20:24], s1)

			if _, err := walOut.Write(frame); err != nil {
				return lastFrameNo, fmt.Errorf("failed to write WAL frame: %w", err)
			}

			lastFrameNo = frameNo + i
		}

		if framesInBatch < batchSize {
			break
		}

		frameNo += framesInBatch
	}

	if err := walOut.Sync(); err != nil {
		return lastFrameNo, fmt.Errorf("failed to sync WAL file: %w", err)
	}

	return lastFrameNo, nil
}

func (i *TursoServerClient) ExportMetadata(outputFile string, info *ExportInfo, durableFrameNum int) error {
	out, err := os.Create(outputFile + "-info")
	if err != nil {
		return fmt.Errorf("failed to create info file: %w", err)
	}
	defer out.Close()

	hasher := crc32.New(crc32.MakeTable(crc32.IEEE))
	var versionBytes [4]byte
	var durableFrameNumBytes [4]byte
	var generationBytes [4]byte
	binary.LittleEndian.PutUint32(versionBytes[:], 0)
	binary.LittleEndian.PutUint32(durableFrameNumBytes[:], uint32(durableFrameNum))
	binary.LittleEndian.PutUint32(generationBytes[:], uint32(info.CurrentGeneration))
	hasher.Write(versionBytes[:])
	hasher.Write(durableFrameNumBytes[:])
	hasher.Write(generationBytes[:])
	hash := int(hasher.Sum32())

	metadata := struct {
		Hash            int `json:"hash"`
		Version         int `json:"version"`
		DurableFrameNum int `json:"durable_frame_num"`
		Generation      int `json:"generation"`
	}{
		Hash:            hash,
		Version:         0,
		DurableFrameNum: durableFrameNum,
		Generation:      info.CurrentGeneration,
	}
	if err := json.NewEncoder(out).Encode(metadata); err != nil {
		return fmt.Errorf("failed to write metadata to file: %w", err)
	}

	return nil
}
