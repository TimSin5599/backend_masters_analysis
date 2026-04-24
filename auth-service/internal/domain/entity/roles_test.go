package entity_test

import (
	"testing"

	"auth-service/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"Valid Admin", entity.RoleAdmin, true},
		{"Valid Manager", entity.RoleManager, true},
		{"Valid Expert", entity.RoleExpert, true},
		{"Valid Operator", entity.RoleOperator, true},
		{"Invalid Role User", "user", false},
		{"Empty Role", "", false},
		{"Case Sensitive Admin", "ADMIN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := entity.IsValidRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}
