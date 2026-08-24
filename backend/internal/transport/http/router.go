package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Handlers struct {
	AuthHandler       *AuthHandler
	ProjectHandler    *ProjectHandler
	EndpointHandler   *EndpointHandler
	IngestionHandler  *IngestionHandler
	RequestHandler    *RequestHandler
	SSEHandler        *SSEHandler
	ReplayHandler     *ReplayHandler
	MockHandler       *MockHandler
	AIHandler         *AIHandler
	AlertHandler      *AlertHandler
	ForwardingHandler *ForwardingHandler
}

func SetupRouter(h *Handlers, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "x-organization-id"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "apisentinel-backend-go",
			"version": "0.1.0",
		})
	})

	// Swagger UI Documentation (/docs & /swagger)
	RegisterSwagger(r)

	// Public Webhook Gateway
	r.HandleFunc("/hook/{slug}", h.IngestionHandler.HandleWebhook)

	// API Routes
	r.Route("/api", func(api chi.Router) {
		// Public Auth
		api.Post("/auth/register", h.AuthHandler.Register)
		api.Post("/auth/login", h.AuthHandler.Login)

		// Protected Routes
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(jwtSecret))

			protected.Get("/auth/me", h.AuthHandler.Me)

			// Projects
			protected.With(middleware.RequireTenant).Get("/projects", h.ProjectHandler.List)
			protected.With(middleware.RequireTenant).Post("/projects", h.ProjectHandler.Create)
			protected.With(middleware.RequireTenant).Get("/projects/{id}", h.ProjectHandler.Get)
			protected.With(middleware.RequireTenant).Delete("/projects/{id}", h.ProjectHandler.Delete)

			// Endpoints & Mocks & Forwarding
			protected.With(middleware.RequireTenant).Get("/projects/{projectId}/endpoints", h.EndpointHandler.List)
			protected.With(middleware.RequireTenant).Post("/projects/{projectId}/endpoints", h.EndpointHandler.Create)
			protected.With(middleware.RequireTenant).Get("/endpoints/{endpointId}/mocks", h.MockHandler.List)
			protected.With(middleware.RequireTenant).Post("/endpoints/{endpointId}/mocks", h.MockHandler.Create)
			protected.With(middleware.RequireTenant).Post("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.SaveConfig)
			protected.With(middleware.RequireTenant).Get("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.GetConfig)
			protected.With(middleware.RequireTenant).Get("/endpoints/{endpointId}/dlq", h.ForwardingHandler.ListDLQ)

			// Multi-Channel Alerting
			protected.With(middleware.RequireTenant).Post("/projects/{projectId}/alerts", h.AlertHandler.CreateChannel)
			protected.With(middleware.RequireTenant).Get("/projects/{projectId}/alerts", h.AlertHandler.ListChannels)
			protected.With(middleware.RequireTenant).Delete("/alerts/{id}", h.AlertHandler.DeleteChannel)
			protected.With(middleware.RequireTenant).Post("/alerts/{id}/test", h.AlertHandler.SendTestAlert)

			// Requests & Replay
			protected.With(middleware.RequireTenant).Get("/projects/{projectId}/requests", h.RequestHandler.ListByProject)
			protected.With(middleware.RequireTenant).Post("/requests/{id}/replay", h.ReplayHandler.Execute)
			protected.With(middleware.RequireTenant).Get("/projects/{projectId}/replays", h.ReplayHandler.ListByProject)

			// AI Finding Explanations
			protected.Post("/ai/explain", h.AIHandler.ExplainFinding)

			// Realtime SSE Stream
			protected.Get("/projects/{projectId}/events/stream", h.SSEHandler.Stream)
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
