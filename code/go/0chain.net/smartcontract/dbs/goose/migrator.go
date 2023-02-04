package goose

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func Init() {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
}

func Migrate(db *sql.DB) {
	if err := goose.Up(db, "migrations"); err != nil {
		panic(err)
	}
}
