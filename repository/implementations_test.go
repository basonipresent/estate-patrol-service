package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SawitProRecruitment/UserService/droneplan"
	"github.com/SawitProRecruitment/UserService/treestats"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("connection reset")

func newMockRepo(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	})
	return &Repository{Db: db}, mock
}

func TestCreateEstate(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("returns generated id", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`INSERT INTO estate`).WithArgs(5, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

		out, err := repo.CreateEstate(ctx, CreateEstateInput{Length: 5, Width: 1})
		require.NoError(t, err)
		require.Equal(t, id, out.Id)
	})

	t.Run("wraps db error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`INSERT INTO estate`).WillReturnError(errDB)

		_, err := repo.CreateEstate(ctx, CreateEstateInput{Length: 5, Width: 1})
		require.ErrorIs(t, err, errDB)
	})
}

func TestGetEstate(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("found", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT id, length, width FROM estate`).WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"id", "length", "width"}).AddRow(id.String(), 5, 1))

		out, err := repo.GetEstate(ctx, GetEstateInput{Id: id})
		require.NoError(t, err)
		require.Equal(t, GetEstateOutput{Id: id, Length: 5, Width: 1}, out)
	})

	t.Run("no rows is ErrNotFound", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT id, length, width FROM estate`).WillReturnError(sql.ErrNoRows)

		_, err := repo.GetEstate(ctx, GetEstateInput{Id: id})
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("wraps db error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT id, length, width FROM estate`).WillReturnError(errDB)

		_, err := repo.GetEstate(ctx, GetEstateInput{Id: id})
		require.ErrorIs(t, err, errDB)
		require.NotErrorIs(t, err, ErrNotFound)
	})
}

func TestCreateTree(t *testing.T) {
	ctx := context.Background()
	estateID, treeID := uuid.New(), uuid.New()
	input := CreateTreeInput{EstateId: estateID, X: 2, Y: 1, Height: 10}

	t.Run("returns generated id", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`INSERT INTO tree`).WithArgs(estateID, 2, 1, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(treeID.String()))

		out, err := repo.CreateTree(ctx, input)
		require.NoError(t, err)
		require.Equal(t, treeID, out.Id)
	})

	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "unique violation is ErrPlotOccupied", dbErr: &pq.Error{Code: pgUniqueViolation}, wantErr: ErrPlotOccupied},
		{name: "foreign key violation is ErrNotFound", dbErr: &pq.Error{Code: pgForeignKeyViolation}, wantErr: ErrNotFound},
		{name: "other pq error is wrapped", dbErr: &pq.Error{Code: "23514"}, wantErr: &pq.Error{Code: "23514"}},
		{name: "other error is wrapped", dbErr: errDB, wantErr: errDB},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			mock.ExpectQuery(`INSERT INTO tree`).WillReturnError(tc.dbErr)

			_, err := repo.CreateTree(ctx, input)
			if pqWant, ok := tc.wantErr.(*pq.Error); ok {
				var pqGot *pq.Error
				require.ErrorAs(t, err, &pqGot)
				require.Equal(t, pqWant.Code, pqGot.Code)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestListTrees(t *testing.T) {
	ctx := context.Background()
	estateID := uuid.New()
	cols := []string{"x", "y", "height"}

	t.Run("returns trees", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT x, y, height FROM tree`).WithArgs(estateID).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(2, 1, 5).AddRow(3, 1, 3))

		out, err := repo.ListTrees(ctx, ListTreesInput{EstateId: estateID})
		require.NoError(t, err)
		require.Equal(t, []droneplan.Tree{{X: 2, Y: 1, Height: 5}, {X: 3, Y: 1, Height: 3}}, out.Trees)
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT x, y, height FROM tree`).WillReturnError(errDB)

		_, err := repo.ListTrees(ctx, ListTreesInput{EstateId: estateID})
		require.ErrorIs(t, err, errDB)
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT x, y, height FROM tree`).
			WillReturnRows(sqlmock.NewRows(cols).AddRow("not-a-number", 1, 5))

		_, err := repo.ListTrees(ctx, ListTreesInput{EstateId: estateID})
		require.Error(t, err)
	})

	t.Run("error while iterating", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT x, y, height FROM tree`).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(2, 1, 5).RowError(0, errDB))

		_, err := repo.ListTrees(ctx, ListTreesInput{EstateId: estateID})
		require.ErrorIs(t, err, errDB)
	})
}

func TestGetHeightHistogram(t *testing.T) {
	ctx := context.Background()
	estateID := uuid.New()
	cols := []string{"height", "count"}

	t.Run("returns buckets", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT height, count\(\*\) FROM tree`).WithArgs(estateID).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(5, 2).AddRow(3, 1))

		out, err := repo.GetHeightHistogram(ctx, GetHeightHistogramInput{EstateId: estateID})
		require.NoError(t, err)
		require.Equal(t, []treestats.HeightCount{{Height: 5, Count: 2}, {Height: 3, Count: 1}}, out.Histogram)
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT height`).WillReturnError(errDB)

		_, err := repo.GetHeightHistogram(ctx, GetHeightHistogramInput{EstateId: estateID})
		require.ErrorIs(t, err, errDB)
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT height`).
			WillReturnRows(sqlmock.NewRows(cols).AddRow("tall", 1))

		_, err := repo.GetHeightHistogram(ctx, GetHeightHistogramInput{EstateId: estateID})
		require.Error(t, err)
	})

	t.Run("error while iterating", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(`SELECT height`).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(5, 2).RowError(0, errDB))

		_, err := repo.GetHeightHistogram(ctx, GetHeightHistogramInput{EstateId: estateID})
		require.ErrorIs(t, err, errDB)
	})
}
