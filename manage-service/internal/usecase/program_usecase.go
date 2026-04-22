package usecase

import (
	"context"
	"manage-service/internal/domain/entity"
)

type ProgramUseCase struct {
	repo ProgramRepo
}

func NewProgramUseCase(repo ProgramRepo) *ProgramUseCase {
	return &ProgramUseCase{repo: repo}
}

func (uc *ProgramUseCase) ListPrograms(ctx context.Context) ([]entity.Program, error) {
	return uc.repo.ListPrograms(ctx)
}

func (uc *ProgramUseCase) GetProgramByID(ctx context.Context, id int64) (entity.Program, error) {
	return uc.repo.GetProgramByID(ctx, id)
}

func (uc *ProgramUseCase) CreateProgram(ctx context.Context, p entity.Program) (entity.Program, error) {
	if p.Status == "" {
		p.Status = "active"
	}
	return uc.repo.CreateProgram(ctx, p)
}

func (uc *ProgramUseCase) UpdateProgramStatus(ctx context.Context, id int64, status string) error {
	return uc.repo.UpdateProgramStatus(ctx, id, status)
}
