package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"manage-service/internal/domain/entity"
	"mime/multipart"
	"net/http"
	"time"
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
	// 5-minute timeout for data extraction
	extractCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
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
	// doc.FileType holds "personal_data", "diploma", etc.
	err = writer.WriteField("category", doc.FileType)
	if err != nil {
		return nil, fmt.Errorf("failed to write category field: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(extractCtx, "POST", c.baseURL+"/extract", body)
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
	var rawResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		fmt.Printf("[EXTRACTION] ❌ Failed to decode response: %v\n", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make(map[string]string)
	for k, v := range rawResult {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			// Convert non-string types (structs, arrays, numbers) to JSON string
			b, _ := json.Marshal(val)
			result[k] = string(b)
		}
	}

	fmt.Printf("[EXTRACTION] ✅ Success! Extracted %d fields\n", len(result))
	return result, nil
}

func (c *HTTPClient) ClassifyDocument(ctx context.Context, fileName string, content []byte) (string, []string, error) {
	// 2-minute timeout for classification
	classifyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	fmt.Printf("[CLASSIFICATION] Triggering for %s | Size: %d bytes\n", fileName, len(content))

	// Prepare multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, bytes.NewReader(content))
	if err != nil {
		return "", nil, fmt.Errorf("failed to copy content: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return "", nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(classifyCtx, "POST", c.baseURL+"/v1/classify", body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("classification service returned status: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Category string   `json:"category"`
		Error    string   `json:"error"`
		Warnings []string `json:"warnings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return "", result.Warnings, fmt.Errorf("classification error: %s", result.Error)
	}

	fmt.Printf("[CLASSIFICATION] ✅ Result: %s\n", result.Category)
	return result.Category, result.Warnings, nil
}
