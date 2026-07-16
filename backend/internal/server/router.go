package server

import (
	"fmt"
	"net/http"

	"passo/backend/internal/account"
	"passo/backend/internal/admin"
	"passo/backend/internal/apikey"
	"passo/backend/internal/auth"
	"passo/backend/internal/chat"
	"passo/backend/internal/guardrail"
	"passo/backend/internal/notify"
	"passo/backend/internal/progress"
	"passo/backend/internal/roadmap"
	"passo/backend/internal/subscription"
)

// Router creates and returns a ServeMux with all application routes.
func Router() *http.ServeMux {
	mux := http.NewServeMux()

	// Global middleware: logging (can be extended)
	middleware := guardrail.Middleware
	authMiddleware := guardrail.AuthMiddleware

	// Auth routes (public)
	mux.HandleFunc("POST /auth/signup", middleware(auth.HandleSignUp))
	mux.HandleFunc("POST /auth/login", middleware(auth.HandleLogin))
	mux.HandleFunc("POST /auth/refresh", middleware(auth.HandleRefresh))
	mux.HandleFunc("POST /auth/logout", middleware(authMiddleware(auth.HandleLogout)))

	// API key routes (require auth)
	mux.HandleFunc("POST /apikey", middleware(authMiddleware(apikey.HandleStoreKey)))
	mux.HandleFunc("GET /apikey", middleware(authMiddleware(apikey.HandleGetKey))) // optional

	// Roadmap routes (require auth)
	mux.HandleFunc("POST /roadmap/generate", middleware(authMiddleware(roadmap.HandleGenerate)))
	mux.HandleFunc("POST /roadmap/replan", middleware(authMiddleware(roadmap.HandleReplan)))

	// Progress routes (require auth)
	mux.HandleFunc("PATCH /progress/{task_id}", middleware(authMiddleware(progress.HandleUpdateTask)))

	// Chat routes (require auth)
	mux.HandleFunc("POST /chat", middleware(authMiddleware(chat.HandleChat)))

	// Notification routes (require auth)
	mux.HandleFunc("POST /push/subscribe", middleware(authMiddleware(notify.HandleSubscribe)))
	mux.HandleFunc("POST /push/vapid/publickey", middleware(authMiddleware(notify.HandleVAPIDPublicKey)))

	// Subscription routes (require auth)
	mux.HandleFunc("POST /subscription/start", middleware(authMiddleware(subscription.HandleStart)))

	// Admin routes (require auth + internal admin check inside handler)
	mux.HandleFunc("GET /admin/dashboard/growth", middleware(authMiddleware(admin.HandleGrowth)))
	mux.HandleFunc("GET /admin/dashboard/engagement", middleware(authMiddleware(admin.HandleEngagement)))
	mux.HandleFunc("GET /admin/dashboard/roadmap", middleware(authMiddleware(admin.HandleRoadmapHealth)))
	mux.HandleFunc("GET /admin/dashboard/ai", middleware(authMiddleware(admin.HandleAIReliability)))
	mux.HandleFunc("GET /admin/dashboard/monetization", middleware(authMiddleware(admin.HandleMonetization)))
	mux.HandleFunc("GET /admin/dashboard/support", middleware(authMiddleware(admin.HandleSupport)))

	// Account routes (require auth)
	mux.HandleFunc("DELETE /account", middleware(authMiddleware(account.HandleDelete)))
	mux.HandleFunc("GET /account/export", middleware(authMiddleware(account.HandleExport)))

	// Health check (public)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Health check endpoint hit")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}
