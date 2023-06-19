package model

import "time"

type Card struct {
	ID           int
	UserID       int
	StripeCardID string
	CreatedAt    time.Time
}
