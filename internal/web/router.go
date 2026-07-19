package web

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/web/handlers"
)

func NewRouter(cfg *config.Config, db *gorm.DB, session *scs.SessionManager) http.Handler {
	r := chi.NewRouter()

	// Base middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Session middleware
	r.Use(session.LoadAndSave)

	// CSRF middleware
	// Set secure option false for development, but in production it should be true.
	csrfMiddleware := csrf.Protect(
		[]byte(cfg.CSRFAuthKey),
		csrf.Secure(false), // Disable Secure flag for localhost dev (HTTP)
		csrf.FieldName("csrf_token"),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.TrustedOrigins([]string{"localhost:8080", "127.0.0.1:8080"}),
	)
	r.Use(csrfMiddleware)

	// Static Files Handler
	fileServer(r, "/static", http.Dir("ui"))

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(db, session, cfg)
	profileHandler := handlers.NewProfileHandler(db, session)
	ordersHandler := handlers.NewOrdersHandler(db, session)
	homeHandler := handlers.NewHomeHandler(db, session)

	// Routes
	r.Get("/", homeHandler.Show)

	// Auth Routes
	r.Get("/auth/github", authHandler.Login)
	r.Get("/auth/github/callback", authHandler.Callback)
	r.Post("/auth/logout", authHandler.Logout)

	// Profile Routes
	r.Get("/profile", profileHandler.Show)
	r.Post("/profile/role", profileHandler.SwitchRole)
	r.Post("/profile/update", profileHandler.Update)
	r.Post("/profile/sync", profileHandler.SyncGitHub)

	// Orders Routes
	r.Get("/orders", ordersHandler.List)
	r.Get("/orders/new", ordersHandler.CreateForm)
	r.Post("/orders", ordersHandler.Create)
	r.Get("/orders/{id}", ordersHandler.Detail)
	r.Post("/orders/{id}/bids", ordersHandler.CreateBid)
	r.Post("/orders/{id}/bids/{bidId}/accept", ordersHandler.AcceptBid)
	r.Post("/orders/{id}/cancel", ordersHandler.CancelOrder)
	r.Post("/orders/{id}/chat/send", ordersHandler.SendChatMessage)
	r.Get("/orders/{id}/chat/messages", ordersHandler.GetChatMessages)

	return r
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))

	r.Get(path+"/*", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
