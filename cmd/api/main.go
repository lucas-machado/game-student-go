package main

import (
	"errors"
	"game-student-go/internal/database"
	"game-student-go/internal/notifications"
	"github.com/sendgrid/sendgrid-go"
	log "github.com/sirupsen/logrus"
	"net/http"
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

	sender := notifications.NewSender(sendgrid.NewSendClient(cfg.SendgridAPIKey))

	server := NewServer(port, db, cfg.JWTKey, sender)

	if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
