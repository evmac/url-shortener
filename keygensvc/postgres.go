package main

import (
	"context"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"log"
)

const PgErrCodeUniqueViolation = "23505"

type PostgresDb interface {
	Refresh()
	connect() error
	queryInt(sql string, params ...interface{}) (int, error)
	close()
}

type postgresDb struct {
	connStr string
	Conn *pgx.Conn
}

func NewPostgresDb(connStr string) PostgresDb {
	return &postgresDb{connStr: connStr}
}

func isMigrationNoChangeError(err error) bool {
	return err == migrate.ErrNoChange
}

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == PgErrCodeUniqueViolation
	}
	return false
}

func (db postgresDb) Refresh() {
	m, err := migrate.New("file://db/migrations", db.connStr)
	if err != nil {
		log.Fatalf("Error initiating migrations: %s", err)
	}
	if downErr := m.Down(); downErr != nil && !isMigrationNoChangeError(downErr) {
		log.Fatalf("Error running migrations down: %s", downErr)
	}
	if upErr := m.Up(); upErr != nil && !isMigrationNoChangeError(upErr) {
		log.Fatalf("Error running migrations up: %s", upErr)
	}
}

func (db *postgresDb) connect() error {
	conn, err := pgx.Connect(context.Background(), db.connStr)
	if err != nil {
		log.Printf("Error connecting to Postgres database: %s", err)
		return err
	}

	db.Conn = conn
	return nil
}

func (db *postgresDb) queryInt(sql string, params ...interface{}) (int, error) {
	var receiver int
	err := db.Conn.QueryRow(context.Background(), sql, params...).Scan(&receiver)
	if err != nil {
		log.Printf("Error running query returning int: %s", err)
		return 0, err
	}
	return receiver, nil
}

func (db *postgresDb) close() {
	if db.Conn != nil {
		db.Conn.Close(context.Background())
		db.Conn = nil
	}
}
