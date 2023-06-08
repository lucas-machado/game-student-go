package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupDatabase(t *testing.T) Client {
	c, err := NewClient("user=ps_user password=ps_password dbname=backend sslmode=disable host=localhost")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		if _, err = c.(*client).db.Exec("DELETE FROM users"); err != nil {
			t.Fatalf("Failed to clean up users table: %v", err)
		}

		if _, err = c.(*client).db.Exec("DELETE FROM courses"); err != nil {
			t.Fatalf("Failed to clean up users table: %v", err)
		}

		c.Close()
	})

	if _, err = c.(*client).db.Exec("DELETE FROM users"); err != nil {
		t.Fatalf("Failed to clean up users table: %v", err)
	}

	if _, err = c.(*client).db.Exec("DELETE FROM courses"); err != nil {
		t.Fatalf("Failed to clean up users table: %v", err)
	}

	return c
}

func TestConnect(t *testing.T) {
	db := setupDatabase(t)
	assert.NotNil(t, db)
}

func TestCreateUser(t *testing.T) {
	db := setupDatabase(t)

	// User to create
	email := "TestUser@test.com"
	password := "TestPassword"

	// Create the user
	user, err := db.CreateUser(email, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	assert.Equal(t, email, user.Email)
}

func TestGetUsers(t *testing.T) {
	db := setupDatabase(t)

	// User to create
	email := "TestUser@test.com"
	password := "TestPassword"

	// Create the user
	_, err := db.CreateUser(email, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Now let's try to fetch the users from the database
	users, err := db.GetUsers()
	if err != nil {
		t.Fatalf("Failed to fetch users: %v", err)
	}

	// Check if the user was correctly inserted
	found := false
	for _, user := range users {
		if user.Email == email {
			found = true
			break
		}
	}

	// Check if the user was found
	assert.True(t, found, "The user was not found in the database")
}

func TestGetCourses(t *testing.T) {
	db := setupDatabase(t)

	if _, err := db.(*client).db.Exec("INSERT INTO courses (name, description, logo_url) VALUES ('Intro to Programming', 'A beginner course for programming.', 'http://example.com/logo.png')"); err != nil {
		t.Fatalf("Failed to clean up users table: %v", err)
	}

	// Assuming some courses exist in the database
	courses, err := db.GetCourses()
	if err != nil {
		t.Fatalf("Failed to fetch courses: %v", err)
	}

	// Assert that at least one course is returned
	assert.True(t, len(courses) > 0, "No courses were found in the database")
}

func TestGetCourseByID(t *testing.T) {
	db := setupDatabase(t)

	row := db.(*client).db.QueryRow("INSERT INTO courses (name, description, logo_url) VALUES ('Intro to Programming', 'A beginner course for programming.', 'http://example.com/logo.png') RETURNING id")
	var id int
	if err := row.Scan(&id); err != nil {
		t.Fatalf("Failed to retrieve id: %v", err)
	}

	// Fetch the course
	course, err := db.GetCourseByID(id)
	if err != nil {
		t.Fatalf("Failed to fetch course by ID: %v", err)
	}

	// Assert that the course has the correct ID
	assert.Equal(t, id, course.ID)
}
