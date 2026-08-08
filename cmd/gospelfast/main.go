package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gospelfast/gospelfast/internal/db"
)

var (
	dbURL   = envOrDefault("DATABASE_URL", "postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable")
	port    = envOrDefault("PORT", "8080")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	database, err := db.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	var count int
	if err := database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM translations").Scan(&count); err != nil {
		log.Printf("warning: could not check translations: %v", err)
	}
	if count == 0 {
		log.Println("No translations found. Run 'gospelfast-cli seed' or 'gospelfast-cli import' to load texts.")
	} else {
		log.Printf("Found %d translation(s)", count)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Gospelfast server starting on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	server.Shutdown(context.Background())
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
