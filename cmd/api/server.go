package main

import (
	"encoding/json"
	"fmt"
	"game-student-go/internal/database"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type Server struct {
	port int
	db   database.Client
}

func NewServer(port int, db database.Client) *Server {
	return &Server{
		port: port,
		db:   db,
	}
}

func (s *Server) Run() error {
	router := mux.NewRouter()

	router.HandleFunc("/users", s.createUser).Methods("POST")
	router.HandleFunc("/users", s.ListUsers).Methods("GET")

	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", s.port), router)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.db.CreateUser(request.Email, request.Password)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"id": strconv.Itoa(user.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
