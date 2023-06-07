package main

import (
	"game-student-go/internal/database"
	log "github.com/sirupsen/logrus"
	"strconv"
)

func main() {
	log.Println("starting game student server")

	cfg, err := ReadConfig()
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		log.Fatalf("converting port to integer: %v", err)
	}

	db, err := database.NewClient(cfg.DBCon)
	if err != nil {
		log.Fatalf("creating database client: %v", err)
	}
	defer db.Close()

	server := NewServer(port, db)

	log.Fatal(server.Run())
}
