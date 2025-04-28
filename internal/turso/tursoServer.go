package turso

import (
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
func (i *TursoServerClient) UploadFile(filepath string, onUploadProgress func(progressPct int, uploadedBytes int64, totalBytes int64, elapsedTime time.Duration, done bool)) error {
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

	// Send POST request with streaming body
	r, err := i.client.PostBinary("/v1/upload", progressTracker)
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

func (i *TursoServerClient) Export(outputFile string, withMetadata bool) error {
	res, err := i.client.Get("/info", nil)
	if err != nil {
		return fmt.Errorf("failed to fetch database info: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("database software version does not support exporting")
	}
	if res.StatusCode != http.StatusOK {
		return parseResponseError(res)
	}
	var info struct {
		CurrentGeneration int `json:"current_generation"`
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to decode /info response: %w", err)
	}

	exportRes, err := i.client.Get(fmt.Sprintf("/export/%d", info.CurrentGeneration), nil)
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

	if withMetadata {
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
		binary.LittleEndian.PutUint32(durableFrameNumBytes[:], 0)
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
			DurableFrameNum: 0,
			Generation:      info.CurrentGeneration,
		}
		if err := json.NewEncoder(out).Encode(metadata); err != nil {
			return fmt.Errorf("failed to write metadata to file: %w", err)
		}
	}
	return nil
}
