package model

import "time"

type Charge struct {
	ID             int
	StripeChargeID string
	StripeCardID   string
	UserID         int
	Amount         int64
	Currency       string
	Description    string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
