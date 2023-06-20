package database

import (
	"database/sql"
	"fmt"
	"game-student-go/internal/model"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v74"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Client interface {
	Close()
	CreateUser(email, password, stripeId string) (model.User, error)
	GetUsers() ([]model.User, error)
	GetUserByEmail(email string) (model.User, error)
	GetUserByID(id int) (model.User, error)
	GetCourses() ([]model.Course, error)
	GetCourseByID(id int) (model.Course, error)
	GetTrainingByID(id int) (model.Training, error)
	AddCard(userID int, stripePayMethodID string) (*model.Card, error)
	GetCard(cardID int) (*model.Card, error)
	AddPayment(pi *stripe.PaymentIntent, userID int) (*model.Payment, error)
	GetPayment(paymentID string) (*model.Payment, error)
	UpdatePaymentStatus(payment *model.Payment) (*model.Payment, error)
}

type client struct {
	db *sql.DB
}

func NewClient(connStr string) (Client, error) {
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return &client{db: db}, nil
}

func (c *client) CreateUser(email, password, stripeId string) (model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("hashing password: %w", err)
	}

	query := `INSERT INTO users (email, password, stripe_id) VALUES ($1, $2, $3) RETURNING id, email, stripe_id`
	var user model.User
	err = c.db.QueryRow(query, email, hashedPassword, stripeId).Scan(&user.ID, &user.Email, &user.StripeId)
	if err != nil {
		return model.User{}, fmt.Errorf("executing user insert and returning data: %w", err)
	}

	return user, nil
}

func (c *client) GetUsers() ([]model.User, error) {
	rows, err := c.db.Query("SELECT ID, email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (c *client) Close() {
	err := c.db.Close()
	if err != nil {
		log.Errorf("closing database: %v", err)
	}
}

func (c *client) GetUserByEmail(email string) (model.User, error) {
	query := `SELECT id, email, password FROM users WHERE email = $1`
	var user model.User
	err := c.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, fmt.Errorf("no user found with email: %s", email)
		}
		return model.User{}, fmt.Errorf("querying for user by email: %w", err)
	}

	return user, nil
}

func (c *client) GetUserByID(id int) (model.User, error) {
	query := `SELECT id, email FROM users WHERE id = $1`
	var user model.User
	err := c.db.QueryRow(query, id).Scan(&user.ID, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, fmt.Errorf("no user found with id: %d", id)
		}
		return model.User{}, fmt.Errorf("querying for user by id: %w", err)
	}

	return user, nil
}

func (c *client) GetCourses() ([]model.Course, error) {
	rows, err := c.db.Query("SELECT ID, name, description, logo_url FROM courses")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var course model.Course
		if err := rows.Scan(&course.ID, &course.Name, &course.Description, &course.LogoURL); err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}

	return courses, nil
}

func (c *client) GetCourseByID(id int) (model.Course, error) {
	query := `SELECT id, name, description, logo_url FROM courses WHERE id = $1`
	var course model.Course
	err := c.db.QueryRow(query, id).Scan(&course.ID, &course.Name, &course.Description, &course.LogoURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Course{}, fmt.Errorf("no course found with id: %v", id)
		}
		return model.Course{}, fmt.Errorf("querying for course by id: %w", err)
	}

	return course, nil
}

func (c *client) GetTrainingByID(id int) (model.Training, error) {
	query := `SELECT id, sequence, topic, name, url, is_free, project_url, course_id FROM trainings WHERE id = $1`
	var training model.Training
	err := c.db.QueryRow(query, id).Scan(&training.ID, &training.Sequence, &training.Topic, &training.Name, &training.URL, &training.IsFree, &training.ProjectURL, &training.CourseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.Training{}, fmt.Errorf("no training found with id: %v", id)
		}
		return model.Training{}, fmt.Errorf("querying for training by id: %w", err)
	}

	return training, nil
}

func (c *client) AddCard(userID int, stripePayMethodID string) (*model.Card, error) {
	var card model.Card

	err := c.db.QueryRow(
		`INSERT INTO cards (user_id, stripe_pay_method_id, created_at) 
         VALUES ($1, $2, $3)
         RETURNING id, user_id, stripe_pay_method_id, created_at`,
		userID,
		stripePayMethodID,
		time.Now(),
	).Scan(&card.ID, &card.UserID, &card.StripePayMethodID, &card.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("unable to add card: %w", err)
	}

	return &card, nil
}

func (c *client) GetCard(cardID int) (*model.Card, error) {
	var card model.Card

	err := c.db.QueryRow(
		"SELECT id, user_id, stripe_pay_method_id, created_at FROM cards WHERE id = $1",
		cardID,
	).Scan(&card.ID, &card.UserID, &card.StripePayMethodID, &card.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// No card with this ID was found
			return nil, nil
		}

		return nil, fmt.Errorf("unable to get card: %w", err)
	}

	return &card, nil
}

func (c *client) AddPayment(pi *stripe.PaymentIntent, userID int) (*model.Payment, error) {
	newPayment := &model.Payment{
		StripePaymentIntentID: pi.ID,
		StripePayMethodID:     pi.PaymentMethod.ID,
		UserID:                userID,
		Amount:                pi.Amount,
		Currency:              string(pi.Currency),
		Status:                string(pi.Status),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	query := `
				INSERT INTO payments (stripe_payment_intent_id, stripe_pay_method_id, user_id, amount, currency, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id
			`

	err := c.db.QueryRow(
		query,
		newPayment.StripePaymentIntentID,
		newPayment.StripePayMethodID,
		newPayment.UserID,
		newPayment.Amount,
		newPayment.Currency,
		newPayment.Status,
		newPayment.CreatedAt,
		newPayment.UpdatedAt,
	).Scan(&newPayment.ID)

	if err != nil {
		return nil, fmt.Errorf("unable to add payment: %w", err)
	}

	return newPayment, nil
}

func (c *client) GetPayment(paymentID string) (*model.Payment, error) {
	var payment model.Payment

	err := c.db.QueryRow(
		`SELECT id, stripe_payment_intent_id, user_id, amount, currency, status, created_at, updated_at 
         FROM payments 
         WHERE stripe_payment_intent_id = $1`,
		paymentID,
	).Scan(&payment.ID, &payment.StripePaymentIntentID, &payment.UserID, &payment.Amount, &payment.Currency, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no payment found with ID: %s", paymentID)
		}

		return nil, fmt.Errorf("unable to get payment: %w", err)
	}

	return &payment, nil
}

func (c *client) UpdatePaymentStatus(payment *model.Payment) (*model.Payment, error) {
	query := `
		UPDATE payments 
		SET status = $1, updated_at = $2 
		WHERE stripe_payment_intent_id = $3
		RETURNING id, stripe_payment_intent_id, amount, status, created_at, updated_at
	`

	err := c.db.QueryRow(
		query,
		payment.Status,
		time.Now(),
		payment.StripePaymentIntentID,
	).Scan(
		&payment.ID,
		&payment.StripePaymentIntentID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("unable to update payment: %w", err)
	}

	return payment, nil
}
