package main

import (
	"context"
	"encoding/json"
	"fmt"
	"game-student-go/internal/database"
	"game-student-go/internal/notifications"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	db          database.Client
	jwtKey      string
	newRelicApp *newrelic.Application
	sender      *notifications.Sender
	http.Server
}

type JWTClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func NewServer(port int, db database.Client, jwtKey string, newRelicApp *newrelic.Application, sender *notifications.Sender) *Server {
	s := &Server{
		db:          db,
		jwtKey:      jwtKey,
		newRelicApp: newRelicApp,
		sender:      sender,
	}
	s.Addr = fmt.Sprintf("0.0.0.0:%d", port)
	return s
}

func (s *Server) Run() error {
	router := mux.NewRouter()

	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/users", s.createUser)).Methods("POST")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/signin", s.Signin)).Methods("POST")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/users/{id}", s.authenticate(s.GetUserByID))).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/courses", s.getCourses)).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/courses/{id}", s.getCourseByID)).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/trainings/{id}", s.getTrainingByID)).Methods("GET")

	s.Handler = router

	log.Printf("listening requests at %v", s.Addr)

	return s.ListenAndServe()
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest
	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(request.Email),
	}
	stripeCustomer, err := customer.New(params)
	if err != nil {
		http.Error(w, "Failed to create Stripe customer", http.StatusInternalServerError)
		return
	}

	user, err := s.db.CreateUser(request.Email, request.Password, stripeCustomer.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.sender.SendRegistrationEmail(request.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	err = json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userId, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByID(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

func (s *Server) getCourses(w http.ResponseWriter, _ *http.Request) {
	courses, err := s.db.GetCourses()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(courses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) getCourseByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid course ID format", http.StatusBadRequest)
		return
	}

	course, err := s.db.GetCourseByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(course)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) getTrainingByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Invalid training ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid training ID format", http.StatusBadRequest)
		return
	}

	training, err := s.db.GetTrainingByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse, err := json.Marshal(training)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
