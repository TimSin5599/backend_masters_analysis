package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"manage-service/internal/domain/entity"
)

type ProgramRepo struct {
	pool *pgxpool.Pool
}

func NewProgramRepo(pool *pgxpool.Pool) *ProgramRepo {
	return &ProgramRepo{pool: pool}
}

func (r *ProgramRepo) ListPrograms(ctx context.Context) ([]entity.Program, error) {
	query := `SELECT id, title, year, description, status, created_at FROM programs ORDER BY year DESC, title ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []entity.Program
	for rows.Next() {
		var p entity.Program
		err = rows.Scan(&p.ID, &p.Title, &p.Year, &p.Description, &p.Status, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

func (r *ProgramRepo) GetProgramByID(ctx context.Context, id int64) (entity.Program, error) {
	query := `SELECT id, title, year, description, status, created_at FROM programs WHERE id=$1`
	var p entity.Program
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Title, &p.Year, &p.Description, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *ProgramRepo) CreateProgram(ctx context.Context, p entity.Program) (entity.Program, error) {
	query := `INSERT INTO programs (title, year, description, status)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id, title, year, description, status, created_at`
	err := r.pool.QueryRow(ctx, query, p.Title, p.Year, p.Description, p.Status).
		Scan(&p.ID, &p.Title, &p.Year, &p.Description, &p.Status, &p.CreatedAt)
	return p, err
}

func (r *ProgramRepo) UpdateProgramStatus(ctx context.Context, id int64, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE programs SET status=$1 WHERE id=$2`, status, id)
	return err
}
