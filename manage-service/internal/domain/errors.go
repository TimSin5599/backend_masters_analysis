package domain

import "errors"

var (
	ErrEvaluationImmutable = errors.New("evaluation is already completed and cannot be modified")
	ErrScoreExceedsMax     = errors.New("score exceeds maximum allowed value for this criteria")
	ErrExpertMissing       = errors.New("expert not found or not assigned to slot")
	ErrApplicantNotFound   = errors.New("applicant not found")
)
