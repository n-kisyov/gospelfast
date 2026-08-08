package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gospelfast/gospelfast/internal/admin"
	"github.com/gospelfast/gospelfast/internal/api"
	"github.com/gospelfast/gospelfast/internal/cache"
	"github.com/gospelfast/gospelfast/internal/db"

	_ "github.com/gospelfast/gospelfast/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

var (
	dbURL    = envOrDefault("DATABASE_URL", "postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable")
	redisURL = envOrDefault("REDIS_URL", "localhost:6379")
	port     = envOrDefault("PORT", "8080")
)

// @title           Gospelfast API
// @version         1.0
// @description     Fast full-text search Bible API
// @host            localhost:8080
// @BasePath        /
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

	var c *cache.Cache
	c, err = cache.New(redisURL)
	if err != nil {
		log.Printf("redis not available, running without cache: %v", err)
		c = nil
	} else {
		defer c.Close()
	}

	apiHandler := api.NewHandler(database, c)
	adminHandler := admin.NewHandler(database, c)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Public API
	mux.HandleFunc("GET /api/translations", apiHandler.ListTranslations)
	mux.HandleFunc("GET /api/books", apiHandler.ListBooks)
	mux.HandleFunc("GET /api/books/{book}", apiHandler.GetBook)
	mux.HandleFunc("GET /api/chapters/{t}/{chapter}", apiHandler.GetChapter)
	mux.HandleFunc("GET /api/verses", apiHandler.GetVerses)
	mux.HandleFunc("GET /api/search", apiHandler.Search)

	// Admin API (basic auth)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /admin/api/translations", adminHandler.ListTranslations)
	adminMux.HandleFunc("DELETE /admin/api/translations/{id}", adminHandler.DeleteTranslation)
	mux.Handle("/admin/", admin.AuthMiddleware(adminMux))

	// Swagger
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	wrapped := withLogging(withCORS(mux))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      wrapped,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Gospelfast server starting on :%s", port)
		log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	shutdownCtx, sdCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer sdCancel()
	server.Shutdown(shutdownCtx)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
