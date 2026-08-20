package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	importpkg "github.com/gospelfast/gospelfast/internal/import"
	"github.com/gospelfast/gospelfast/internal/web"

	_ "github.com/gospelfast/gospelfast/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/crypto/bcrypt"
)

var (
	dbURL        = envOrDefault("DATABASE_URL", "postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable")
	redisURL     = envOrDefault("REDIS_URL", "localhost:6379")
	port         = envOrDefault("PORT", "8080")
	adminPass    = os.Getenv("ADMIN_PASSWORD")
	templatesDir = envOrDefault("TEMPLATES_DIR", "web/templates")
	staticDir    = envOrDefault("STATIC_DIR", "web/static")
)

// @title           Gospelfast API
// @version         1.0
// @description     Fast full-text search Bible API
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

	userCount, _ := database.UserCount(ctx)
	if userCount == 0 {
		generated := false
		if adminPass == "" {
			adminPass = generateRandomPassword()
			generated = true
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err == nil {
			_, err := database.CreateUser(ctx, "admin", string(hash), "admin")
			if err != nil {
				log.Printf("failed to seed admin user: %v", err)
			} else if generated {
				log.Println("=====================================================")
				log.Println("No ADMIN_PASSWORD set. Generated one-time admin login:")
				log.Println("  username: admin")
				log.Printf("  password: %s\n", adminPass)
				log.Println("Set ADMIN_PASSWORD to control this on future deploys.")
				log.Println("=====================================================")
			} else {
				log.Println("Admin user created (username: admin)")
			}
		}
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
	adminAPIHandler := admin.NewHandler(database, c)

	pipeline := importpkg.New(database)
	jobManager := admin.NewJobManager(pipeline)

	webHandler, err := web.NewHandler(database, c, templatesDir)
	if err != nil {
		log.Fatalf("web handler: %v", err)
	}

	adminWebHandler, err := admin.NewWebHandler(database, c, jobManager, templatesDir)
	if err != nil {
		log.Fatalf("admin web handler: %v", err)
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(staticDir))))

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Auth
	mux.HandleFunc("GET /login", admin.HandleLogin(database))
	mux.HandleFunc("POST /login", admin.HandleLogin(database))
	mux.HandleFunc("POST /logout", admin.HandleLogout(database))

	// Web pages
	mux.HandleFunc("GET /{$}", webHandler.Home)
	mux.HandleFunc("GET /search", webHandler.Search)
	mux.HandleFunc("GET /reader", webHandler.Reader)
	mux.HandleFunc("GET /reader/chapter", webHandler.ReaderChapter)
	mux.HandleFunc("GET /compare", webHandler.Compare)
	mux.HandleFunc("GET /compare/results", webHandler.CompareResults)
	mux.HandleFunc("GET /genbook", webHandler.Genbook)

	// Public API
	mux.HandleFunc("GET /api/translations", apiHandler.ListTranslations)
	mux.HandleFunc("GET /api/books", apiHandler.ListBooks)
	mux.HandleFunc("GET /api/books/{book}", apiHandler.GetBook)
	mux.HandleFunc("GET /api/chapters/{t}/{chapter}", apiHandler.GetChapter)
	mux.HandleFunc("GET /api/verses", apiHandler.GetVerses)
	mux.HandleFunc("GET /api/search", apiHandler.Search)
	mux.HandleFunc("GET /api/genbooks", apiHandler.ListGenbooks)
	mux.HandleFunc("GET /api/genbooks/entry", apiHandler.GetGenbookEntry)
	mux.HandleFunc("GET /api/commentary", apiHandler.GetCommentary)

	// Admin pages (requires login)
	mux.Handle("GET /admin", admin.RequireAdmin(adminWebHandler.Dashboard))
	mux.Handle("GET /admin/api/imports/{id}", admin.RequireAdmin(adminWebHandler.ImportStatus))
	mux.Handle("POST /admin/api/imports", admin.RequireAdmin(adminWebHandler.StartImport))
	mux.Handle("GET /admin/api/translations", admin.RequireAdmin(adminAPIHandler.ListTranslations))
	mux.Handle("DELETE /admin/api/translations/{id}", admin.RequireAdmin(adminAPIHandler.DeleteTranslation))
	mux.Handle("GET /admin/api/users", admin.RequireAdmin(adminWebHandler.ListUsers))
	mux.Handle("POST /admin/api/users", admin.RequireAdmin(adminWebHandler.CreateUser))
	mux.Handle("DELETE /admin/api/users/{id}", admin.RequireAdmin(adminWebHandler.DeleteUser))
	mux.Handle("PUT /admin/api/users/{id}/password", admin.RequireAdmin(adminWebHandler.UpdateUserPassword))

	// Swagger
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Catch-all: only reached when no other pattern above matched the
	// request at all, so it's safe to render the pretty 404 page here.
	// (Handlers that legitimately return a 404 for a matched route, e.g.
	// "translation not found", write their own complete response and
	// must NOT be touched again after the fact.)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webHandler.RenderError(w, http.StatusNotFound, "Page not found", "The page you were looking for does not exist.")
	})

	wrapped := admin.SessionMiddleware(database)(
		withRecovery(withLogging(withCORS(mux))),
	)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      wrapped,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Gospelfast server starting on :%s", port)
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

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func generateRandomPassword() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is effectively unreachable in practice; fall
		// back to a fixed-but-loud placeholder rather than panicking.
		return "CHANGE-ME-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
