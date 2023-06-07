package main

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	log.SetLevel(log.InfoLevel)
	log.Println("starting migrate")

	dbConn := os.Getenv("DB_CONN")

	if dbConn == "" {
		dbConn = "user=ps_user password=ps_password dbname=backend sslmode=disable host=0.0.0.0"
	}

	log.Println("Connecting to db using conn:", dbConn)

	db, err := sql.Open("postgres", dbConn)

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Errorf("closing the db: %v", err)
		}
	}(db)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://./migrations"),
		"postgres", driver)

	if err != nil {
		log.Fatal(err)
	}

	// Apply the migrations
	err = m.Up()
	if err != nil {
		if err != migrate.ErrNoChange {
			log.Fatal(err)
		}
	}

	fmt.Println("Migrations complete!")
}
