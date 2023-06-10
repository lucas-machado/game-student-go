package main

import (
	"errors"
	"fmt"
	"github.com/ardanlabs/conf"
)

type Config struct {
	Port            string `conf:"default:8080,env:PORT"`
	DBCon           string `conf:"default:user=ps_user password=ps_password dbname=backend sslmode=disable host=localhost,env:DB_CONN"`
	JWTKey          string `conf:"default:your_secret_key,env:JWT_KEY"`
	NewRelicAppName string `conf:"default:game-student-go,env:NEW_RELIC_APP_NAME"`
	NewRelicLicense string `conf:"env:NEW_RELIC_LICENSE"`
	SendgridAPIKey  string `conf:"env:SENDGRID_API_KEY"`
}

func ReadConfig() (*Config, error) {
	var cfg Config
	help, err := conf.ParseOSArgs("APP", &cfg)

	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
