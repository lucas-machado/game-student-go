package model

type Course struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
}
