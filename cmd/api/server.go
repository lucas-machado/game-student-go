package main

import (
	"context"
	"encoding/json"
	"fmt"
	"game-student-go/internal/database"
	"game-student-go/internal/model"
	"game-student-go/internal/notifications"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/ephemeralkey"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/paymentmethod"
	"github.com/stripe/stripe-go/v74/setupintent"
	"golang.org/x/crypto/bcrypt"
	"io"
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
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/users/{id}/card", s.authenticate(s.addCard))).Methods("POST")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/users/{id}/cards", s.listCards)).Methods("GET")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/users/{id}/cards/{paym_id}/authorize", s.authenticate(s.authorizePayment))).Methods("POST")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/payment/{payment_id}/capture", s.authenticate(s.captureFunds))).Methods("POST")
	router.HandleFunc(newrelic.WrapHandleFunc(s.newRelicApp, "/stripe/webhook", s.handleStripeWebhook)).Methods("POST")

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
		log.Error("Failed to create Stripe customer:", err)
		http.Error(w, "Failed to create Stripe customer", http.StatusInternalServerError)
		return
	}

	user, err := s.db.CreateUser(request.Email, request.Password, stripeCustomer.ID)
	if err != nil {
		log.Error("Failed to create user:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.sender.SendRegistrationEmail(request.Email)
	if err != nil {
		log.Error("Failed to send registration email:", err)
		http.Error(w, "Failed to send registration email", http.StatusInternalServerError)
		return
	}

	response := CreateUserResponse{
		ID: strconv.Itoa(user.ID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response:", err)
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

func (s *Server) addCard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Bad Request - User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Bad Request - User ID must be an integer", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Bad Request - User not found", http.StatusBadRequest)
		return
	}

	params := &stripe.EphemeralKeyParams{
		Customer: stripe.String(user.StripeId),
	}
	params.AddMetadata("user_id", userIDStr)

	ephemeralKey, err := ephemeralkey.New(params)
	if err != nil {
		http.Error(w, "Failed to create ephemeral key", http.StatusInternalServerError)
		return
	}

	intentParams := &stripe.SetupIntentParams{
		Customer: stripe.String(user.StripeId),
	}

	intent, err := setupintent.New(intentParams)
	if err != nil {
		http.Error(w, "Failed to create setup intent", http.StatusInternalServerError)
		return
	}

	response := struct {
		EphemeralKeyID     string `json:"ephemeral_key_id"`
		IntentClientSecret string `json:"intent_client_secret"`
	}{
		EphemeralKeyID:     ephemeralKey.ID,
		IntentClientSecret: intent.ClientSecret,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) listCards(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the URL parameters
	vars := mux.Vars(r)
	userIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Bad Request - User ID is required", http.StatusBadRequest)
		return
	}

	// Convert the user ID to an integer
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Bad Request - User ID must be an integer", http.StatusBadRequest)
		return
	}

	// Get the user from the database
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Bad Request - User not found", http.StatusBadRequest)
		return
	}

	// Get the Stripe customer ID
	customerID := user.StripeId

	// List all cards for the Stripe customer
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(customerID),
		Type:     stripe.String(string(stripe.PaymentMethodTypeCard)),
	}
	result := paymentmethod.List(params)

	var cards []model.Card
	for result.Next() {
		pm := result.PaymentMethod()
		card := model.Card{
			ID:       pm.ID,
			Brand:    string(pm.Card.Brand),
			LastFour: pm.Card.Last4,
			ExpMonth: uint64(pm.Card.ExpMonth),
			ExpYear:  uint64(pm.Card.ExpYear),
		}
		cards = append(cards, card)
	}
	if err := result.Err(); err != nil {
		http.Error(w, "Error retrieving cards", http.StatusInternalServerError)
		return
	}

	// Encode and return the cards
	if err := json.NewEncoder(w).Encode(cards); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) authorizePayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Bad Request - User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Bad Request - User ID must be an integer", http.StatusBadRequest)
		return
	}

	payMethodID := vars["paym_id"]

	var request ChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate the application fee (20% of the total amount)
	appFee := int64(float64(request.Amount) * 0.20)

	params := &stripe.PaymentIntentParams{
		Amount:               stripe.Int64(request.Amount),
		Currency:             stripe.String(request.Currency),
		PaymentMethod:        stripe.String(payMethodID),
		Confirm:              stripe.Bool(true),
		ConfirmationMethod:   stripe.String(string(stripe.PaymentIntentConfirmationMethodManual)),
		CaptureMethod:        stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
		ApplicationFeeAmount: stripe.Int64(appFee), // Set an application fee
		TransferData: &stripe.PaymentIntentTransferDataParams{
			Destination: stripe.String("{CONNECTED_STRIPE_ACCOUNT_ID}"), // The ID of the connected account
		},
	}
	pi, err := paymentintent.New(params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s.db.AddPayment(pi, userID)
	if err != nil {
		http.Error(w, "Error storing charge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if pi.Status == stripe.PaymentIntentStatusRequiresCapture {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ClientSecret string `json:"client_secret"`
	}{
		ClientSecret: pi.ClientSecret,
	})
}

func (s *Server) captureFunds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Extract the PaymentIntent ID from the URL
	paymentID := vars["payment_id"]

	// Get the payment from the database
	payment, err := s.db.GetPayment(paymentID)
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	// Retrieve the PaymentIntent from Stripe
	pi, err := paymentintent.Get(payment.StripePaymentIntentID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the PaymentIntent is still in a capturable status
	if pi.Status != stripe.PaymentIntentStatusRequiresCapture {
		http.Error(w, "PaymentIntent cannot be captured", http.StatusBadRequest)
		return
	}

	// Capture the PaymentIntent
	params := &stripe.PaymentIntentCaptureParams{
		AmountToCapture: stripe.Int64(payment.Amount),
	}

	_, err = paymentintent.Capture(pi.ID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.captured":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			http.Error(w, "Error parsing webhook JSON", http.StatusBadRequest)
			return
		}

		payment, err := s.db.GetPayment(paymentIntent.ID)

		if err != nil {
			http.Error(w, "Payment not found", http.StatusBadRequest)
			return
		}

		payment.Status = string(paymentIntent.Status)

		_, err = s.db.UpdatePaymentStatus(payment)
		if err != nil {
			http.Error(w, "Payment status updated", http.StatusBadRequest)
			return
		}

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			http.Error(w, "Error parsing webhook JSON", http.StatusBadRequest)
			return
		}
		fmt.Printf("PaymentIntent failed!\n")
	default:
		fmt.Fprintf(w, "Unhandled event type: %s\n", event.Type)
		return
	}

	w.WriteHeader(http.StatusOK)
}
