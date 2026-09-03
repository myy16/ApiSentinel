package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Handlers struct {
	AuthHandler            *AuthHandler
	ProjectHandler         *ProjectHandler
	EndpointHandler        *EndpointHandler
	IngestionHandler       *IngestionHandler
	RequestHandler         *RequestHandler
	SSEHandler             *SSEHandler
	ReplayHandler          *ReplayHandler
	MockHandler            *MockHandler
	AIHandler              *AIHandler
	AlertHandler           *AlertHandler
	ForwardingHandler      *ForwardingHandler
	FindingHandler         *FindingHandler
	APIKeyHandler          *APIKeyHandler
	AgentHandler           *AgentHandler
	WebhookSecurityHandler *WebhookSecurityHandler
	DeliveryHandler        *DeliveryHandler
	TemplateHandler        *TemplateHandler
	SchemaHandler          *SchemaHandler
	TestSuiteHandler       *TestSuiteHandler
}

func SetupRouter(h *Handlers, jwtSecret string, queries *database.Queries, corsOrigin string) *chi.Mux {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS Setup — support credentials for cookie-based auth
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001"}
	allowCredentials := true
	if corsOrigin != "" && corsOrigin != "*" {
		allowedOrigins = strings.Split(corsOrigin, ",")
		for i, o := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(o)
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	// Root redirect to API documentation (/docs)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
	})

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
	requireOwner := middleware.RequireRole("OWNER")
	requireDeveloper := middleware.RequireRole("DEVELOPER")

	projectGuard := middleware.RequireProjectOwnership(queries, "projectId")
	endpointGuard := middleware.RequireEndpointOwnership(queries, "endpointId")
	requestGuard := middleware.RequireRequestOwnership(queries, "id")
	alertGuard := middleware.RequireAlertChannelOwnership(queries, "id")
	dlqGuard := middleware.RequireDLQRecordOwnership(queries, "id")

	// API Routes
	r.Route("/api", func(api chi.Router) {
		// Public Auth
		api.Post("/auth/register", h.AuthHandler.Register)
		api.Post("/auth/login", h.AuthHandler.Login)
		api.Post("/auth/logout", h.AuthHandler.Logout)
		api.Post("/auth/refresh", h.AuthHandler.Refresh)

		// Protected Routes
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(jwtSecret))

			protected.Get("/auth/me", h.AuthHandler.Me)

			// Projects
			protected.With(tenantGuard).Get("/projects", h.ProjectHandler.List)
			protected.With(tenantGuard, requireDeveloper).Post("/projects", h.ProjectHandler.Create)
			protected.With(tenantGuard, middleware.RequireProjectOwnership(queries, "id")).Get("/projects/{id}", h.ProjectHandler.Get)
			protected.With(tenantGuard, middleware.RequireProjectOwnership(queries, "id"), requireDeveloper).Put("/projects/{id}", h.ProjectHandler.Update)
			protected.With(tenantGuard, middleware.RequireProjectOwnership(queries, "id"), requireOwner).Delete("/projects/{id}", h.ProjectHandler.Delete)

			// Endpoints & Mocks & Forwarding & Schemas
			protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/endpoints", h.EndpointHandler.List)
			protected.With(tenantGuard, projectGuard, requireDeveloper).Post("/projects/{projectId}/endpoints", h.EndpointHandler.Create)
			protected.With(tenantGuard, projectGuard, endpointGuard, requireDeveloper).Put("/projects/{projectId}/endpoints/{endpointId}", h.EndpointHandler.Update)
			protected.With(tenantGuard, projectGuard, endpointGuard, requireOwner).Delete("/projects/{projectId}/endpoints/{endpointId}", h.EndpointHandler.Delete)
			protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/mocks", h.MockHandler.List)
			protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/mocks", h.MockHandler.Create)
			protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.SaveConfig)
			protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/forwarding", h.ForwardingHandler.GetConfig)
			protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/dlq", h.ForwardingHandler.ListDLQ)
			protected.With(tenantGuard, endpointGuard, requireOwner).Delete("/endpoints/{endpointId}/dlq", h.ForwardingHandler.PurgeDLQ)
			protected.With(tenantGuard, dlqGuard, requireDeveloper).Post("/dlq/{id}/retry", h.ForwardingHandler.RetryDLQ)
			protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/schema", h.EndpointHandler.SaveSchema)
			protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/schema", h.EndpointHandler.GetSchema)
			protected.With(tenantGuard, endpointGuard, requireDeveloper).Delete("/endpoints/{endpointId}/schema", h.EndpointHandler.DeleteSchema)
			if h.WebhookSecurityHandler != nil {
				protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/webhook-security", h.WebhookSecurityHandler.Get)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Put("/endpoints/{endpointId}/webhook-security", h.WebhookSecurityHandler.Save)
				protected.With(tenantGuard, endpointGuard, requireOwner).Delete("/endpoints/{endpointId}/webhook-security", h.WebhookSecurityHandler.Delete)
			}

			// Multi-Channel Alerting
			protected.With(tenantGuard, projectGuard, requireDeveloper).Post("/projects/{projectId}/alerts", h.AlertHandler.CreateChannel)
			protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/alerts", h.AlertHandler.ListChannels)
			protected.With(tenantGuard, alertGuard, requireDeveloper).Delete("/alerts/{id}", h.AlertHandler.DeleteChannel)
			protected.With(tenantGuard, alertGuard, requireDeveloper).Post("/alerts/{id}/test", h.AlertHandler.SendTestAlert)

			// Requests & Replay (Milestone 11)
			protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/requests", h.RequestHandler.ListByProject)
			protected.With(tenantGuard, requestGuard, requireDeveloper).Post("/requests/{id}/replay", h.ReplayHandler.Execute)
			protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/replays", h.ReplayHandler.ListByProject)
			protected.With(tenantGuard).Get("/replays/{id}", h.ReplayHandler.GetReplay)

			// Replay Test Suites & Scenario Runner (Milestone 12)
			if h.TestSuiteHandler != nil {
				protected.With(tenantGuard, projectGuard, requireDeveloper).Post("/projects/{projectId}/test-suites", h.TestSuiteHandler.Create)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/test-suites", h.TestSuiteHandler.ListByProject)
				protected.With(tenantGuard).Get("/test-suites/{id}", h.TestSuiteHandler.Get)
				protected.With(tenantGuard, requireDeveloper).Delete("/test-suites/{id}", h.TestSuiteHandler.Delete)
				protected.With(tenantGuard, requireDeveloper).Post("/test-suites/{id}/run", h.TestSuiteHandler.Run)
			}

			// Security Findings (Real DB & Statistics) & Agent Scans
			if h.FindingHandler != nil {
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/findings", h.FindingHandler.ListByProject)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/findings/stats", h.FindingHandler.GetStats)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/scans", h.FindingHandler.ListScansByProject)
			}

			// AI Finding Explanations
			protected.With(tenantGuard).Post("/ai/explain", h.AIHandler.ExplainFinding)

			// API Keys (Multi-Key Management & Rotation)
			if h.APIKeyHandler != nil {
				protected.With(tenantGuard, projectGuard, requireDeveloper).Post("/projects/{projectId}/keys", h.APIKeyHandler.Create)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/keys", h.APIKeyHandler.List)
				protected.With(tenantGuard, projectGuard, requireDeveloper).Delete("/projects/{projectId}/keys/{keyId}", h.APIKeyHandler.Revoke)
			}

			// Realtime SSE Stream (Guarded by tenant membership)
			protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/events/stream", h.SSEHandler.Stream)

			// Delivery Control Plane (Milestone 3 & Faz 1)
			if h.DeliveryHandler != nil {
				protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/deliveries", h.DeliveryHandler.ListByEndpoint)
				protected.With(tenantGuard).Get("/deliveries/{id}/timeline", h.DeliveryHandler.GetTimeline)
				protected.With(tenantGuard, requireDeveloper).Post("/deliveries/{id}/replay", h.DeliveryHandler.Replay)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/delivery-kpis", h.DeliveryHandler.GetKPIs)
				protected.With(tenantGuard, projectGuard).Get("/projects/{projectId}/audit-logs", h.DeliveryHandler.ListAuditLogs)
			}

			// Agent Sessions (Real gRPC connected agents)
			if h.AgentHandler != nil {
				protected.With(tenantGuard).Get("/agents/sessions", h.AgentHandler.ListSessions)
			}

			// Provider Templates Catalog (Milestone 7)
			if h.TemplateHandler != nil {
				protected.Get("/templates/providers", h.TemplateHandler.ListProviders)
			}

			// Schema Baselines & Contracts (Milestone 9 & Milestone 10)
			if h.SchemaHandler != nil {
				protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/schemas", h.SchemaHandler.ListBaselines)
				protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/schemas/active", h.SchemaHandler.GetActiveBaseline)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/schemas", h.SchemaHandler.SaveManual)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/schemas/infer", h.SchemaHandler.InferBaseline)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/schemas/openapi", h.SchemaHandler.ImportOpenAPI)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Put("/endpoints/{endpointId}/schemas/{schemaId}/activate", h.SchemaHandler.ActivateBaseline)

				// Schema Drift Endpoints (Milestone 10)
				protected.With(tenantGuard, endpointGuard).Get("/endpoints/{endpointId}/drifts", h.SchemaHandler.ListDrifts)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/drifts/{driftId}/accept", h.SchemaHandler.AcceptDrift)
				protected.With(tenantGuard, endpointGuard, requireDeveloper).Post("/endpoints/{endpointId}/drifts/{driftId}/dismiss", h.SchemaHandler.DismissDrift)
			}
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
