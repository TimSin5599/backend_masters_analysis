package usecase_test

import (
	"context"
	"testing"

	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"

	"github.com/stretchr/testify/assert"
)

type mockProgramRepo struct {
	ListProgramsFunc        func(ctx context.Context) ([]entity.Program, error)
	GetProgramByIDFunc      func(ctx context.Context, id int64) (entity.Program, error)
	CreateProgramFunc       func(ctx context.Context, p entity.Program) (entity.Program, error)
	UpdateProgramStatusFunc func(ctx context.Context, id int64, status string) error
}

func (m *mockProgramRepo) ListPrograms(ctx context.Context) ([]entity.Program, error) {
	return m.ListProgramsFunc(ctx)
}

func (m *mockProgramRepo) GetProgramByID(ctx context.Context, id int64) (entity.Program, error) {
	return m.GetProgramByIDFunc(ctx, id)
}

func (m *mockProgramRepo) CreateProgram(ctx context.Context, p entity.Program) (entity.Program, error) {
	return m.CreateProgramFunc(ctx, p)
}

func (m *mockProgramRepo) UpdateProgramStatus(ctx context.Context, id int64, status string) error {
	return m.UpdateProgramStatusFunc(ctx, id, status)
}

func TestProgramUseCase_CreateProgram(t *testing.T) {
	repo := &mockProgramRepo{
		CreateProgramFunc: func(ctx context.Context, p entity.Program) (entity.Program, error) {
			p.ID = 1
			return p, nil
		},
	}

	uc := usecase.NewProgramUseCase(repo)

	p := entity.Program{Title: "Test Program", Year: 2024}
	created, err := uc.CreateProgram(context.Background(), p)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), created.ID)
	assert.Equal(t, "active", created.Status) // testing the logic `if p.Status == "" { p.Status = "active" }`
}

func TestProgramUseCase_GetProgramByID(t *testing.T) {
	repo := &mockProgramRepo{
		GetProgramByIDFunc: func(ctx context.Context, id int64) (entity.Program, error) {
			return entity.Program{ID: id, Title: "Test Program"}, nil
		},
	}

	uc := usecase.NewProgramUseCase(repo)

	p, err := uc.GetProgramByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)
	assert.Equal(t, "Test Program", p.Title)
}
