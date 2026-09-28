package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"github.com/PedroJ03/go-inventory-management/internal/application"
	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

func newLocationRepositoryForTest(t *testing.T) (*LocationRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewLocationRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func TestLocationRepositoryCreateReturnsGeneratedLocation(t *testing.T) {
	repository, mock, closeDB := newLocationRepositoryForTest(t)
	defer closeDB()
	mock.ExpectQuery(regexp.QuoteMeta(insertLocationSQL)).WithArgs("Warehouse", "Main Street 1", "Springfield").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "street", "city", "active"}).AddRow(7, "Warehouse", "Main Street 1", "Springfield", true),
	)

	got, err := repository.Create(context.Background(), domain.Location{Name: "Warehouse", Street: "Main Street 1", City: "Springfield"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	want := domain.Location{ID: 7, Name: "Warehouse", Street: "Main Street 1", City: "Springfield", Active: true}
	if got != want {
		t.Fatalf("Create() = %+v, want %+v", got, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLocationRepositoryFindByIDMapsRow(t *testing.T) {
	repository, mock, closeDB := newLocationRepositoryForTest(t)
	defer closeDB()
	mock.ExpectQuery(regexp.QuoteMeta(findLocationByIDSQL)).WithArgs(int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "street", "city", "active"}).AddRow(7, "Warehouse", "Main Street 1", "Springfield", false),
	)

	got, err := repository.FindByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if !(!got.Active && got.ID == 7) {
		t.Fatalf("FindByID() = %+v, want inactive location 7", got)
	}
}

func TestLocationRepositoryStatusUpdates(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		call       func(*LocationRepository) (domain.Location, error)
		wantActive bool
	}{
		{name: "activate", query: activateLocationSQL, call: func(r *LocationRepository) (domain.Location, error) { return r.Activate(context.Background(), 7) }, wantActive: true},
		{name: "deactivate", query: deactivateLocationSQL, call: func(r *LocationRepository) (domain.Location, error) { return r.Deactivate(context.Background(), 7) }, wantActive: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, mock, closeDB := newLocationRepositoryForTest(t)
			defer closeDB()
			mock.ExpectQuery(regexp.QuoteMeta(tt.query)).WithArgs(int64(7)).WillReturnRows(
				sqlmock.NewRows([]string{"id", "name", "street", "city", "active"}).AddRow(7, "Warehouse", "Main Street 1", "Springfield", tt.wantActive),
			)
			got, err := tt.call(repository)
			if err != nil || got.Active != tt.wantActive {
				t.Fatalf("status update = %+v, error = %v", got, err)
			}
		})
	}
}

func TestLocationRepositoryMapsNotFound(t *testing.T) {
	repository, mock, closeDB := newLocationRepositoryForTest(t)
	defer closeDB()
	mock.ExpectQuery(regexp.QuoteMeta(findLocationByIDSQL)).WithArgs(int64(7)).WillReturnError(sql.ErrNoRows)

	_, err := repository.FindByID(context.Background(), 7)
	if !errors.Is(err, application.ErrLocationNotFound) {
		t.Fatalf("FindByID() error = %v, want not found error", err)
	}
}

func TestLocationRepositoryRejectsInvalidInput(t *testing.T) {
	repository, mock, closeDB := newLocationRepositoryForTest(t)
	defer closeDB()

	if _, err := repository.Create(context.Background(), domain.Location{ID: 1, Name: "Warehouse", Street: "Main Street 1", City: "Springfield"}); !errors.Is(err, application.ErrInvalidLocation) {
		t.Fatalf("Create() error = %v, want invalid location error", err)
	}
	if _, err := repository.Create(context.Background(), domain.Location{Name: "", Street: "Main Street 1", City: "Springfield"}); !errors.Is(err, application.ErrInvalidLocation) {
		t.Fatalf("Create() error = %v, want invalid location error", err)
	}
	if _, err := repository.FindByID(context.Background(), 0); !errors.Is(err, application.ErrInvalidLocation) {
		t.Fatalf("FindByID() error = %v, want invalid location error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLocationRepositoryMapsConstraintError(t *testing.T) {
	repository, mock, closeDB := newLocationRepositoryForTest(t)
	defer closeDB()
	mock.ExpectQuery(regexp.QuoteMeta(insertLocationSQL)).WithArgs("Warehouse", "Main Street 1", "Springfield").WillReturnError(&pq.Error{Code: "23514"})

	_, err := repository.Create(context.Background(), domain.Location{Name: "Warehouse", Street: "Main Street 1", City: "Springfield"})
	if !errors.Is(err, application.ErrLocationConstraint) {
		t.Fatalf("Create() error = %v, want constraint error", err)
	}
}
