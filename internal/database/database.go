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
	GetUserByID(id int) (model.User, error)
	GetCourses() ([]model.Course, error)
	GetCourseByID(id int) (model.Course, error)
	GetTrainingByID(id int) (model.Training, error)
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
