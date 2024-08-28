package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/segmentio/ksuid"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

type URL struct {
	ShortID     string `json:"short_id"`
	OriginalURL string `json:"original_url"`
}

func main() {
	// Load environment variables from .env file
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatalf("Error loading .env file: %v", err)
	// }
	// fmt.Println("DATABASE_URL:", os.Getenv("DATABASE_URL"))
	// Initialize PostgreSQL connection
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"), // e.g., "localhost:6379"
	})
	defer rdb.Close()

	r := chi.NewRouter()
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Change this to your frontend URL for production
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)
	r.Post("/create", createShortURLHandler)
	r.Get("/{shortID}", redirectHandler)

	// Log that the server is starting
	fmt.Println("Server running on port 8080")

	// Start the HTTP server and log any errors
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func createShortURLHandler(w http.ResponseWriter, r *http.Request) {
	var url URL
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if url.ShortID == "" {
		// Generate a unique short ID if not provided
		url.ShortID = ksuid.New().String()[:8]
	}

	newUUID := uuid.New()

	// Insert the new URL into PostgreSQL
	_, err := db.Exec("INSERT INTO url_redirects (id, short_url_id, original_url) VALUES ($1, $2, $3)", newUUID, url.ShortID, url.OriginalURL)
	if err != nil {
		log.Printf("Error inserting URL into PostgreSQL: %v", err)
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	// Cache the new URL in Redis
	err = rdb.Set(ctx, url.ShortID, url.OriginalURL, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to cache URL in Redis: %v", err)
	}

	// Return the created short URL
	json.NewEncoder(w).Encode(url)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")

	// Check Redis cache first
	originalURL, err := rdb.Get(ctx, shortID).Result()
	if err == redis.Nil {
		// Cache miss: fetch from PostgreSQL
		err = db.QueryRow("SELECT original_url FROM url_redirects WHERE short_url_id = $1", shortID).Scan(&originalURL)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		// Cache the result in Redis for future requests
		rdb.Set(ctx, shortID, originalURL, 24*time.Hour)
	} else if err != nil {
		http.Error(w, "Error accessing cache", http.StatusInternalServerError)
		return
	}

	// Log the redirection with the analytics service
	go logRedirection(shortID)

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func logRedirection(shortID string) {
	http.Post(fmt.Sprintf("https://dsandbox.online/api/analytic/log/%s", shortID), "application/json", nil)
}
