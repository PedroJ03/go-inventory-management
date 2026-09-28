package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/PedroJ03/go-inventory-management/internal/application"
	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

const (
	insertLocationSQL     = `INSERT INTO locations (name, street, city) VALUES ($1, $2, $3) RETURNING id, name, street, city, active`
	findLocationByIDSQL   = `SELECT id, name, street, city, active FROM locations WHERE id = $1`
	activateLocationSQL   = `UPDATE locations SET active = TRUE WHERE id = $1 RETURNING id, name, street, city, active`
	deactivateLocationSQL = `UPDATE locations SET active = FALSE WHERE id = $1 RETURNING id, name, street, city, active`
)

// LocationRepository persists locations using explicit PostgreSQL SQL.
type LocationRepository struct {
	db *sql.DB
}

var _ application.LocationRepository = (*LocationRepository)(nil)

func NewLocationRepository(db *sql.DB) (*LocationRepository, error) {
	if db == nil {
		return nil, errors.New("location repository database must not be nil")
	}
	return &LocationRepository{db: db}, nil
}

func (r *LocationRepository) Create(ctx context.Context, location domain.Location) (domain.Location, error) {
	if location.ID != 0 {
		return domain.Location{}, fmt.Errorf("create location: %w: id must be zero", application.ErrInvalidLocation)
	}
	if err := location.Validate(); err != nil {
		return domain.Location{}, fmt.Errorf("create location: %w: %v", application.ErrInvalidLocation, err)
	}

	row := r.db.QueryRowContext(ctx, insertLocationSQL, location.Name, location.Street, location.City)
	created, err := scanLocation(row)
	if err != nil {
		return domain.Location{}, mapLocationWriteError(err)
	}
	return created, nil
}

func (r *LocationRepository) FindByID(ctx context.Context, id int64) (domain.Location, error) {
	return r.findByID(ctx, id, findLocationByIDSQL)
}

func (r *LocationRepository) Activate(ctx context.Context, id int64) (domain.Location, error) {
	return r.findByID(ctx, id, activateLocationSQL)
}

func (r *LocationRepository) Deactivate(ctx context.Context, id int64) (domain.Location, error) {
	return r.findByID(ctx, id, deactivateLocationSQL)
}

func (r *LocationRepository) findByID(ctx context.Context, id int64, query string) (domain.Location, error) {
	if id <= 0 {
		return domain.Location{}, fmt.Errorf("location id must be positive: %w", application.ErrInvalidLocation)
	}

	location, err := scanLocation(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Location{}, application.ErrLocationNotFound
	}
	if err != nil {
		return domain.Location{}, fmt.Errorf("scan location: %w", err)
	}
	return location, nil
}

type locationRowScanner interface {
	Scan(dest ...any) error
}

func scanLocation(row locationRowScanner) (domain.Location, error) {
	var location domain.Location
	if err := row.Scan(&location.ID, &location.Name, &location.Street, &location.City, &location.Active); err != nil {
		return domain.Location{}, err
	}
	if err := location.Validate(); err != nil {
		return domain.Location{}, fmt.Errorf("validate database location: %w", err)
	}
	return location, nil
}

func mapLocationWriteError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return application.ErrLocationNotFound
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23502", "23514", "22P02":
			return fmt.Errorf("%w: %v", application.ErrLocationConstraint, err)
		}
	}
	return fmt.Errorf("persist location: %w", err)
}
