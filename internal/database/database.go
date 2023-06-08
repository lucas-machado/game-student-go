package database

import (
	"database/sql"
	"fmt"
	"game-student-go/internal/model"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Client interface {
	Close()
	CreateUser(email, password string) (model.User, error)
	GetUsers() ([]model.User, error)
	GetUserByEmail(email string) (model.User, error)
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

func (c *client) CreateUser(email, password string) (model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("hashing password: %w", err)
	}

	query := `INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, email`
	var user model.User
	err = c.db.QueryRow(query, email, hashedPassword).Scan(&user.ID, &user.Email)
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
