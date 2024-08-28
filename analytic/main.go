package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/go-redis/redis/v8"
)

var( 	db *sql.DB
		rdb *redis.Client
		ctx = context.Background()
	)

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASS"), // e.g., "localhost:6379"
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
	r.Post("/log/{shortID}", logHandler)
	r.Get("/stats/{shortID}", statsHandler)
	fmt.Println("Running On 8080")
	http.ListenAndServe(":8080", handler)
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")
	newUUID := uuid.New()
	_, err := db.Exec("INSERT INTO url_analytics (id, short_url_id, access_time) VALUES ($1, $2, $3)", newUUID, shortID, time.Now())
	if err != nil {
		log.Printf("Error inserting URL into PostgreSQL: %v", err)
		http.Error(w, "Failed to log redirection", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Logged redirection for %s", shortID)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")
	var count int
	// Check Redis cache first
	count, err := rdb.Get(ctx, "stats-"+shortID).Result()
	if err == redis.Nil {
		// Cache miss: fetch from PostgreSQL
		err := db.QueryRow("SELECT COUNT(*) FROM url_analytics WHERE short_url_id = $1", shortID).Scan(&count)
		if err != nil {
			log.Printf("Error fetching stats: %v", err)
			http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
		}
		// Cache the result in Redis for future requests
		rdb.Set(ctx, "stats-"+shortID, count, 1*time.Hour)
	} else if err != nil {
		http.Error(w, "Error accessing cache", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "URL %s has been accessed %d times", shortID, count)
}
