package config

import (
	"log"
	"os"
)

var (
	jwtSecret  string
	corsOrigin string
	port       string
)

// Init loads and validates required configuration from environment variables.
// Must be called after godotenv.Load().
func Init() {
	jwtSecret = os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required. Add it to your .env file.")
	}

	corsOrigin = os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:4200"
	}

	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
}

func JWTSecret() string  { return jwtSecret }
func CORSOrigin() string { return corsOrigin }
func Port() string       { return port }
