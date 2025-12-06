package turso

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// progressReader is a custom io.Reader that tracks progress of the upload
// and calls the onProgress callback with the progress of the upload.
type progressReader struct {
	reader     io.Reader
	totalSize  int64
	bytesRead  int64
	startTime  time.Time
	onProgress func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)
	lastUpdate int // Last reported progress percentage. Initially -1 to ensure first update is always sent.
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.bytesRead += int64(n)
		progressPct := int(float64(pr.bytesRead) / float64(pr.totalSize) * 100)

		// Only call progress if we've made at least 1% progress or if we're done
		if progressPct > pr.lastUpdate || errors.Is(err, io.EOF) {
			elapsedTime := time.Since(pr.startTime)
			pr.lastUpdate = progressPct
			pr.onProgress(progressPct, pr.bytesRead, pr.totalSize, elapsedTime, errors.Is(err, io.EOF))
		}
	}
	return n, err
}

type TursoServerClient struct {
	tenant string
	client *Client
}

func NewTursoServerClient(baseURL *url.URL, token string, cliVersion string, org string) (TursoServerClient, error) {
	newClient := New(baseURL, token, cliVersion, org)

	return TursoServerClient{
		tenant: org,
		client: newClient,
	}, nil
}

// UploadFile uploads a database file to the Turso server.
// it assumes a SQLite file exists at 'filepath'.
// it streams the file to the server, and calls the onProgress callback with the progress of the upload.
func (i *TursoServerClient) UploadFile(filepath, remoteEncryptionCipher, remoteEncryptionKey string, onUploadProgress func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats for %s: %w", filepath, err)
	}

	totalSize := stat.Size()

	// Create progress tracking reader
	progressTracker := &progressReader{
		reader:     file,
		totalSize:  totalSize,
		onProgress: onUploadProgress,
		startTime:  time.Now(),
		lastUpdate: -1, // Ensure first update is always sent
	}

	headers := map[string]string{}
	if remoteEncryptionCipher != "" && remoteEncryptionKey != "" {
		headers[EncryptionCipherHeader] = remoteEncryptionCipher
		headers[EncryptionKeyHeader] = remoteEncryptionKey
	}

	// Send POST request with streaming body
	r, err := i.client.PostBinary("/v1/upload", progressTracker, headers)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("upload failed with status code %d and error reading response: %v", r.StatusCode, err)
		}
		return fmt.Errorf("upload failed with status code %d: %s", r.StatusCode, string(body))
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
