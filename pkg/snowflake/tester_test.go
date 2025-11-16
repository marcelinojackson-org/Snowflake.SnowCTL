package snowflake

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/snowflakedb/gosnowflake"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
)

func withMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	origOpen := openFunc
	origDSN := dsnFunc
	openFunc = func(driverName, dsn string) (*sql.DB, error) { return db, nil }
	dsnFunc = func(cfg *gosnowflake.Config) (string, error) { return "dsn", nil }
	return mock, func() {
		openFunc = origOpen
		dsnFunc = origDSN
		db.Close()
	}
}

func TestTestConnectionReturnsServerTime(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectPing()
	mock.ExpectQuery("select current_timestamp\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"CURRENT_TIMESTAMP"}).AddRow("2025-01-01T00:00:00Z"))

	ts, err := TestConnection(context.Background(), &config.Context{AuthMethod: "password"}, "secret")
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if ts != "2025-01-01T00:00:00Z" {
		t.Fatalf("unexpected timestamp %s", ts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}

func TestRunQueryReturnsRows(t *testing.T) {
	mock, cleanup := withMockDB(t)
	defer cleanup()

	mock.ExpectQuery("select 1").WillReturnRows(sqlmock.NewRows([]string{"COL1"}).AddRow(1))

	rows, err := RunQuery(context.Background(), &config.Context{AuthMethod: "pat"}, "secret", "select 1")
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(rows) != 1 || rows[0]["COL1"].(int64) != 1 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}
