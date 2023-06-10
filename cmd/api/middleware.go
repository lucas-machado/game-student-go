package main

import (
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

func (s *Server) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		splitToken := strings.Split(tokenHeader, "Bearer ")
		if len(splitToken) != 2 {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		requestToken := splitToken[1]

		token, err := jwt.ParseWithClaims(requestToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.jwtKey), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
