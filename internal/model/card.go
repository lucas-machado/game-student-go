package model

type Card struct {
	ID       string `json:"id"`
	Brand    string `json:"brand"`
	LastFour string `json:"last_four"`
	ExpMonth uint64 `json:"exp_month"`
	ExpYear  uint64 `json:"exp_year"`
}
