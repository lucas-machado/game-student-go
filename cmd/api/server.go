package main

import (
	"encoding/json"
	"fmt"
	"game-student-go/internal/database"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	port   int
	db     database.Client
	jwtKey string
}

type JWTClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func NewServer(port int, db database.Client, jwtKey string) *Server {
	return &Server{
		port:   port,
		db:     db,
		jwtKey: jwtKey,
	}
}

func (s *Server) Run() error {
	router := mux.NewRouter()

	router.HandleFunc("/users", s.createUser).Methods("POST")
	router.HandleFunc("/users", s.ListUsers).Methods("GET")
	router.HandleFunc("/signin", s.Signin).Methods("POST")

	address := "0.0.0.0"

	log.Printf("listening requests at %v:%v", address, s.port)

	return http.ListenAndServe(fmt.Sprintf("%v:%v", address, s.port), router)
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

func (s *Server) Signin(w http.ResponseWriter, r *http.Request) {
	creds := &SignInRequest{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByEmail(creds.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &JWTClaims{
		Email: creds.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtKey))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
