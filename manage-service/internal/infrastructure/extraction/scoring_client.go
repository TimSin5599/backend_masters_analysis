package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"manage-service/internal/domain/entity"
)

// ScoringHTTPClient реализует интерфейс usecase.ScoringClient.
// Вызывает POST /v1/score на data-extraction-service.
type ScoringHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewScoringClient(baseURL string) *ScoringHTTPClient {
	return &ScoringHTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type scoreRequest struct {
	Criteria      []criterionDTO         `json:"criteria"`
	ApplicantData map[string]interface{} `json:"applicant_data"`
}

type criterionDTO struct {
	Code     string `json:"code"`
	Title    string `json:"title"`
	MaxScore int    `json:"max_score"`
	Scheme   string `json:"scheme"`
}

type scoreResponse struct {
	Scores []entity.ScoringResult `json:"scores"`
	Error  string                 `json:"error,omitempty"`
}

func (c *ScoringHTTPClient) ScorePortfolio(ctx context.Context, criteria []entity.EvaluationCriteria, applicantData map[string]interface{}) ([]entity.ScoringResult, error) {
	dtos := make([]criterionDTO, 0, len(criteria))
	for _, cr := range criteria {
		dtos = append(dtos, criterionDTO{
			Code:     cr.Code,
			Title:    cr.Title,
			MaxScore: cr.MaxScore,
			Scheme:   cr.Scheme,
		})
	}

	body, err := json.Marshal(scoreRequest{
		Criteria:      dtos,
		ApplicantData: applicantData,
	})
	if err != nil {
		return nil, fmt.Errorf("ScoringClient: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/score", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ScoringClient: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ScoringClient: do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ScoringClient: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ScoringClient: service returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result scoreResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("ScoringClient: unmarshal response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("ScoringClient: service error: %s", result.Error)
	}

	return result.Scores, nil
}
