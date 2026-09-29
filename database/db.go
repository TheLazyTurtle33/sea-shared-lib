package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/TheLazyTurtle33/sea-shared-lib/cleanup"
	"github.com/TheLazyTurtle33/sea-shared-lib/logger"
	_ "github.com/lib/pq"
)

const (
	host   = "postgres"
	port   = 5432
	user   = "turt"
	dbname = "sea_db"
)

var password = os.Getenv("POSTGRES_PASSWORD")

type Database struct {
	db *sql.DB
}

var instance *Database

func Get() *Database {
	if instance != nil {
		return instance
	}

	psqlInfo := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
		user, password, host, port, dbname)

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		logger.Error("failed to connect to database", err)
		return nil
	}

	instance = &Database{db: db}
	cleanup.RegisterCleaner(&cleaner{})
	return instance

}

func (d *Database) Query(query string, args ...any) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

func (d *Database) Exec(query string, args ...any) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

type cleaner struct {
	cleanup.Cleaner
}

func (c *cleaner) Clean() {
	if instance == nil {
		return
	}
	err := instance.db.Close()
	if err != nil {
		logger.Error("failed to close database connection", err)
	}
}
