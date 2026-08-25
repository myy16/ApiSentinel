package http

import (
	"encoding/json"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/database"
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
	FindingHandler    *FindingHandler
}

func SetupRouter(h *Handlers, jwtSecret string, queries *database.Queries) *chi.Mux {
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

	// Tenant-verified middleware (validates DB membership)
	tenantGuard := middleware.RequireTenant(queries)

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
			protected.With(tenantGuard).Get("/projects", h.ProjectHandler.List)
			protected.With(tenantGuard).Post("/projects", h.ProjectHandler.Create)
			protected.With(tenantGuard).Get("/projects/{id}", h.ProjectHandler.Get)
			protected.With(tenantGuard).Delete("/projects/{id}", h.ProjectHandler.Delete)

			// Endpoints & Mocks & Forwarding & Schemas
			protected.With(tenantGuard).Get("/projects/{projectId}/endpoints", h.EndpointHandler.List)
			protected.With(tenantGuard).Post("/projects/{projectId}/endpoints", h.EndpointHandler.Create)
			protected.With(tenantGuard).Put("/projects/{projectId}/endpoints/{endpointId}", h.EndpointHandler.Update)
			protected.With(tenantGuard).Delete("/projects/{projectId}/endpoints/{endpointId}", h.EndpointHandler.Delete)
			protected.With(tenantGuard).Get("/endpoints/{endpointId}/mocks", h.MockHandler.List)
			protected.With(tenantGuard).Post("/endpoints/{endpointId}/mocks", h.MockHandler.Create)
			protected.With(tenantGuard).Post("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.SaveConfig)
			protected.With(tenantGuard).Get("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.GetConfig)
			protected.With(tenantGuard).Get("/endpoints/{endpointId}/dlq", h.ForwardingHandler.ListDLQ)
			protected.With(tenantGuard).Delete("/endpoints/{endpointId}/dlq", h.ForwardingHandler.PurgeDLQ)
			protected.With(tenantGuard).Post("/dlq/{id}/retry", h.ForwardingHandler.RetryDLQ)
			protected.With(tenantGuard).Post("/endpoints/{endpointId}/schema", h.EndpointHandler.SaveSchema)
			protected.With(tenantGuard).Get("/endpoints/{endpointId}/schema", h.EndpointHandler.GetSchema)
			protected.With(tenantGuard).Delete("/endpoints/{endpointId}/schema", h.EndpointHandler.DeleteSchema)

			// Multi-Channel Alerting
			protected.With(tenantGuard).Post("/projects/{projectId}/alerts", h.AlertHandler.CreateChannel)
			protected.With(tenantGuard).Get("/projects/{projectId}/alerts", h.AlertHandler.ListChannels)
			protected.With(tenantGuard).Delete("/alerts/{id}", h.AlertHandler.DeleteChannel)
			protected.With(tenantGuard).Post("/alerts/{id}/test", h.AlertHandler.SendTestAlert)

			// Requests & Replay
			protected.With(tenantGuard).Get("/projects/{projectId}/requests", h.RequestHandler.ListByProject)
			protected.With(tenantGuard).Post("/requests/{id}/replay", h.ReplayHandler.Execute)
			protected.With(tenantGuard).Get("/projects/{projectId}/replays", h.ReplayHandler.ListByProject)

			// Security Findings (Real DB & Statistics)
			if h.FindingHandler != nil {
				protected.With(tenantGuard).Get("/projects/{projectId}/findings", h.FindingHandler.ListByProject)
				protected.With(tenantGuard).Get("/projects/{projectId}/findings/stats", h.FindingHandler.GetStats)
			}

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
