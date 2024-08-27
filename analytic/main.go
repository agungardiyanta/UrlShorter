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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var db *sql.DB

func main() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatalf("Error loading .env file: %v", err)
	// }
	// // Initialize PostgreSQL connection
	// fmt.Println("DATABASE_URL:", os.Getenv("DATABASE_URL"))
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
	fmt.Println("Running On 8081")
	http.ListenAndServe(":8081", handler)
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
	err := db.QueryRow("SELECT COUNT(*) FROM url_analytics WHERE short_url_id = $1", shortID).Scan(&count)
	if err != nil {
		log.Printf("Error fetching stats: %v", err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "URL %s has been accessed %d times", shortID, count)
}
