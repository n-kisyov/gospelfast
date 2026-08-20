package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gospelfast/gospelfast/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey string

const sessionCookie = "gospelfast_session"
const userCtxKey ctxKey = "user"

func SessionMiddleware(database *db.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, err := database.GetSession(r.Context(), cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil || user.Role != "admin" {
			if strings.Contains(r.Header.Get("Accept"), "text/html") || strings.Contains(r.Header.Get("HX-Request"), "true") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func GetUser(r *http.Request) *db.User {
	u, _ := r.Context().Value(userCtxKey).(*db.User)
	return u
}

func HandleLogin(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(loginPageHTML))
			return
		}

		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")

			user, err := database.GetUserByUsername(r.Context(), username)
			if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(loginPageHTML + `<p class="text-red-500 text-center mt-4">Invalid username or password</p></div></div></main></body></html>`))
				return
			}

			token, err := database.CreateSession(r.Context(), user.ID, time.Now().Add(24*time.Hour))
			if err != nil {
				http.Error(w, "session error", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookie,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Expires:  time.Now().Add(24 * time.Hour),
			})

			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleLogout(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookie); err == nil {
			_ = database.DeleteSession(r.Context(), cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

const loginPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <script>
        // Match the theme chosen elsewhere in the app before first paint,
        // so this page doesn't flash a light background (or ignore dark
        // mode entirely) while the stylesheet/JS is still loading.
        (function () {
            if (localStorage.getItem('dark') === 'true') {
                document.documentElement.classList.add('dark');
            }
        })();
    </script>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gospelfast — Login</title>
    <link href="/static/css/tailwind.css" rel="stylesheet">
</head>
<body class="bg-gray-50 dark:bg-gray-900 min-h-screen flex items-center justify-center">
<main class="max-w-sm w-full px-4">
<div class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
    <h1 class="text-xl font-bold text-center mb-6 text-gray-900 dark:text-white">Gospelfast</h1>
    <form method="post" class="space-y-4">
        <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Username</label>
            <input type="text" name="username" required autofocus
                   class="w-full border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white rounded-lg px-3 py-2 text-sm">
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">Password</label>
            <input type="password" name="password" required
                   class="w-full border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white rounded-lg px-3 py-2 text-sm">
        </div>
        <button type="submit" class="w-full bg-blue-600 text-white py-2 rounded-lg text-sm font-medium hover:bg-blue-700 transition">
            Sign in
        </button>
    </form>
</div>
</main>
</body>
</html>`
