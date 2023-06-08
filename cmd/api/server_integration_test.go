//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"game-student-go/internal/database"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

var server *Server

func cleanupDB() {
	cfg, err := ReadConfig()
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBCon)
	if err != nil {
		log.Fatalf("Error opening connection to the database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM users;")
	if err != nil {
		log.Fatalf("Error cleaning up users table: %v", err)
	}
}

func TestMain(m *testing.M) {
	// Initialization
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

	server = NewServer(port, db, cfg.JWTKey)

	go func() {
		if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// Allow some time for the server to start
	time.Sleep(100 * time.Millisecond)

	// Run the tests
	exitVal := m.Run()

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	os.Exit(exitVal)
}

func TestCreateUser(t *testing.T) {
	cleanupDB()
	requestBody, _ := json.Marshal(CreateUserRequest{Email: "test@example.com", Password: "testpassword"})
	resp, err := http.Post("http://localhost:8080/users", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Could not send POST request: %v", err)
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestSignin(t *testing.T) {
	cleanupDB()

	// First, create a user
	createUserBody, _ := json.Marshal(CreateUserRequest{Email: "my_test_user", Password: "my_test_password"})
	_, err := http.Post("http://localhost:8080/users", "application/json", bytes.NewBuffer(createUserBody))
	if err != nil {
		t.Fatalf("Could not send POST request to create user: %v", err)
	}

	// Then, sign in
	signinBody, _ := json.Marshal(SignInRequest{Email: "my_test_user", Password: "my_test_password"})
	resp, err := http.Post("http://localhost:8080/signin", "application/json", bytes.NewBuffer(signinBody))
	if err != nil {
		t.Fatalf("Could not send POST request to sign in: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetUserByID(t *testing.T) {
	cleanupDB()

	// First, create a user
	createUserBody, _ := json.Marshal(CreateUserRequest{Email: "my_test_user", Password: "my_test_password"})
	createResp, err := http.Post("http://localhost:8080/users", "application/json", bytes.NewBuffer(createUserBody))
	if err != nil {
		t.Fatalf("Could not send POST request to create user: %v", err)
	}

	var createdUser map[string]string
	err = json.NewDecoder(createResp.Body).Decode(&createdUser)
	if err != nil {
		t.Fatalf("Could not decode create user response: %v", err)
	}
	userId := createdUser["id"]

	// Then, sign in
	signinBody, _ := json.Marshal(SignInRequest{Email: "my_test_user", Password: "my_test_password"})
	signinResp, err := http.Post("http://localhost:8080/signin", "application/json", bytes.NewBuffer(signinBody))
	if err != nil {
		t.Fatalf("Could not send POST request to sign in: %v", err)
	}

	var jwt map[string]string
	err = json.NewDecoder(signinResp.Body).Decode(&jwt)
	if err != nil {
		t.Fatalf("Could not decode JWT response: %v", err)
	}

	// Get the user by ID
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/users/"+userId, nil)
	if err != nil {
		t.Fatalf("Could not create GET request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+jwt["token"])

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Could not send GET request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetCourses(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/courses")
	if err != nil {
		t.Fatalf("Could not send GET request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetCourseByID(t *testing.T) {
	cleanupDB()

	cfg, err := ReadConfig()
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBCon)
	if err != nil {
		log.Fatalf("Error opening connection to the database: %v", err)
	}
	defer db.Close()

	row := db.QueryRow("INSERT INTO courses (name, description, logo_url) VALUES ('Intro to Programming', 'A beginner course for programming.', 'http://example.com/logo.png') RETURNING id")
	var id int
	if err := row.Scan(&id); err != nil {
		t.Fatalf("Failed to retrieve id: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/courses/%v", id))
	if err != nil {
		t.Fatalf("Could not send GET request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
