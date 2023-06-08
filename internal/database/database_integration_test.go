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
		c.Close()
	})

	// Clean up the users table
	if _, err = c.(*client).db.Exec("DELETE FROM users"); err != nil {
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
