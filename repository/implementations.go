package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SawitProRecruitment/UserService/droneplan"
	"github.com/SawitProRecruitment/UserService/treestats"
	"github.com/lib/pq"
)

// Postgres SQLSTATE codes we translate into domain errors.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

func (r *Repository) CreateEstate(ctx context.Context, input CreateEstateInput) (output CreateEstateOutput, err error) {
	err = r.Db.QueryRowContext(ctx,
		`INSERT INTO estate (length, width) VALUES ($1, $2) RETURNING id`,
		input.Length, input.Width,
	).Scan(&output.Id)
	if err != nil {
		return output, fmt.Errorf("create estate: %w", err)
	}
	return output, nil
}

func (r *Repository) GetEstate(ctx context.Context, input GetEstateInput) (output GetEstateOutput, err error) {
	err = r.Db.QueryRowContext(ctx,
		`SELECT id, length, width FROM estate WHERE id = $1`,
		input.Id,
	).Scan(&output.Id, &output.Length, &output.Width)
	if errors.Is(err, sql.ErrNoRows) {
		return output, ErrNotFound
	}
	if err != nil {
		return output, fmt.Errorf("get estate: %w", err)
	}
	return output, nil
}

func (r *Repository) CreateTree(ctx context.Context, input CreateTreeInput) (output CreateTreeOutput, err error) {
	err = r.Db.QueryRowContext(ctx,
		`INSERT INTO tree (estate_id, x, y, height) VALUES ($1, $2, $3, $4) RETURNING id`,
		input.EstateId, input.X, input.Y, input.Height,
	).Scan(&output.Id)

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case pgUniqueViolation: // tree_estate_plot_key: the DB is the race-free check
			return output, ErrPlotOccupied
		case pgForeignKeyViolation:
			return output, ErrNotFound
		}
	}
	if err != nil {
		return output, fmt.Errorf("create tree: %w", err)
	}
	return output, nil
}

func (r *Repository) ListTrees(ctx context.Context, input ListTreesInput) (output ListTreesOutput, err error) {
	rows, err := r.Db.QueryContext(ctx,
		`SELECT x, y, height FROM tree WHERE estate_id = $1`,
		input.EstateId,
	)
	if err != nil {
		return output, fmt.Errorf("list trees: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t droneplan.Tree
		if err = rows.Scan(&t.X, &t.Y, &t.Height); err != nil {
			return output, fmt.Errorf("list trees: %w", err)
		}
		output.Trees = append(output.Trees, t)
	}
	if err = rows.Err(); err != nil {
		return output, fmt.Errorf("list trees: %w", err)
	}
	return output, nil
}

// GetHeightHistogram returns at most 30 rows (heights are 1..30),
// however many trees the estate has.
func (r *Repository) GetHeightHistogram(ctx context.Context, input GetHeightHistogramInput) (output GetHeightHistogramOutput, err error) {
	rows, err := r.Db.QueryContext(ctx,
		`SELECT height, count(*) FROM tree WHERE estate_id = $1 GROUP BY height`,
		input.EstateId,
	)
	if err != nil {
		return output, fmt.Errorf("height histogram: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h treestats.HeightCount
		if err = rows.Scan(&h.Height, &h.Count); err != nil {
			return output, fmt.Errorf("height histogram: %w", err)
		}
		output.Histogram = append(output.Histogram, h)
	}
	if err = rows.Err(); err != nil {
		return output, fmt.Errorf("height histogram: %w", err)
	}
	return output, nil
}
