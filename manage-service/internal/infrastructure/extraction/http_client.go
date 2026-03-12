package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"manage-service/internal/entity"
	"mime/multipart"
	"net/http"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *HTTPClient) TriggerExtraction(ctx context.Context, doc entity.Document, content []byte) (map[string]string, error) {
	fmt.Printf("[EXTRACTION] Triggering for doc %d (%s) | Size: %d bytes\n", doc.ID, doc.FileName, len(content))

	// Prepare multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", doc.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to copy content: %w", err)
	}

	// Add category
	// doc.FileType holds "passport", "diploma", etc. (mapped in UseCase)
	err = writer.WriteField("category", doc.FileType)
	if err != nil {
		return nil, fmt.Errorf("failed to write category field: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/extract", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Printf("[EXTRACTION] ❌ Request failed: %v\n", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("[EXTRACTION] ❌ Failed to decode response: %v\n", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("[EXTRACTION] ✅ Success! Extracted %d fields\n", len(result))
	return result, nil
}
