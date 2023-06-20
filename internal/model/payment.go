package model

import "time"

type Payment struct {
	ID                    int
	StripePaymentIntentID string
	StripePayMethodID     string
	UserID                int
	Amount                int64
	Currency              string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
