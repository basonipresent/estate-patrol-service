// This file contains the repository implementation layer.
package repository

import "database/sql"

type Repository struct {
	Db *sql.DB
}

// NewRepositoryOptions takes an open pool; main owns connecting, pool limits
// and closing it.
type NewRepositoryOptions struct {
	Db *sql.DB
}

func NewRepository(opts NewRepositoryOptions) *Repository {
	return &Repository{
		Db: opts.Db,
	}
}
