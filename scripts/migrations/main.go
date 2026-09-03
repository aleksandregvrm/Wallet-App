package main

// Runs the SQL migrations under shared/migrations against Postgres using goose.
// Built into its own image and run as a one-off "migrate" service in
// docker-compose, ahead of any service that depends on the db.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	"go-task-wallet-service/shared/env"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	dir := flag.String("dir", "shared/migrations", "directory containing the migration files")
	flag.Parse()

	command := "up"
	var args []string
	if rest := flag.Args(); len(rest) > 0 {
		command = rest[0]
		args = rest[1:]
	}

	dsn := env.GetString("POSTGRES_DSN", "postgres://go_master:test@localhost:5433/?sslmode=disable")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open postgres connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}

	if err := goose.RunContext(context.Background(), command, db, *dir, args...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}

	fmt.Printf("goose %s completed against %s\n", command, *dir)
}
