package entity

import "time"

type DocumentQueueTask struct {
	ID               string    `json:"id"`
	ApplicantID      int64     `json:"applicant_id"`
	DocumentCategory string    `json:"document_category"`
	FilePath         string    `json:"file_path"`
	Priority         int       `json:"priority"`
	Status           string    `json:"status"`
	ErrorMessage     *string   `json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
