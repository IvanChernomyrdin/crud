package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var DB *sql.DB

func NewConnection(DSN string) (*sql.DB, error) {
	var err error
	DB, err = sql.Open("pgx", DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := Migrations(DB); err != nil {
		return nil, fmt.Errorf("error create migrations: %w", err)
	}
	return DB, nil
}

func CloseConnection(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed close database connection: %w", err)
	}
	return nil
}

func Migrations(DB *sql.DB) error {
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		return nil
	}
	driver, err := postgres.WithInstance(DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create driver migrations: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			// если новых миграций нет, не ошибка
			return nil
		}
		return fmt.Errorf("failed migrations up: %w", err)
	}
	return nil
}

func PingDatabase(db *sql.DB) error {
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}
