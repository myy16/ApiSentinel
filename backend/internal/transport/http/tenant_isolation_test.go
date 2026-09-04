package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantIsolation_StrictResourceOwnership(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping tenant isolation test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping tenant isolation test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	endpointService := service.NewEndpointService(queries)
	alertService := service.NewAlertService(queries, nil)
	forwardingService := service.NewForwardingService(queries, nil)
	ingestionService := service.NewIngestionService(queries, nil, alertService, forwardingService, nil, nil, nil)
	requestService := service.NewRequestService(queries)
	replayService := service.NewReplayService(queries)
	mockService := service.NewMockService(queries)
	findingService := service.NewFindingService(queries)
	apiKeyService := service.NewAPIKeyService(queries)

	handlers := &Handlers{
		AuthHandler:       NewAuthHandler(authService),
		ProjectHandler:    NewProjectHandler(projectService),
		EndpointHandler:   NewEndpointHandler(endpointService),
		IngestionHandler:  NewIngestionHandler(ingestionService),
		RequestHandler:    NewRequestHandler(requestService),
		SSEHandler:        NewSSEHandler(nil),
		ReplayHandler:     NewReplayHandler(replayService),
		MockHandler:       NewMockHandler(mockService),
		AIHandler:         NewAIHandler(ai.NewExplainer("")),
		AlertHandler:      NewAlertHandler(alertService),
		ForwardingHandler: NewForwardingHandler(forwardingService),
		FindingHandler:    NewFindingHandler(findingService),
		APIKeyHandler:     NewAPIKeyHandler(apiKeyService),
	}

	router := SetupRouter(handlers, cfg.JWTSecret, queries, "*")

	// 1. Register User A
	emailA := fmt.Sprintf("tenant_a_%d@apisentinel.dev", time.Now().UnixNano())
	regBodyA, _ := json.Marshal(map[string]string{
		"email":            emailA,
		"password":         "Password123!",
		"organizationName": "Org Alpha",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(regBodyA))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to register User A: %s", w.Body.String())
	}
	var authA service.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &authA)

	// 2. Register User B
	emailB := fmt.Sprintf("tenant_b_%d@apisentinel.dev", time.Now().UnixNano())
	regBodyB, _ := json.Marshal(map[string]string{
		"email":            emailB,
		"password":         "Password123!",
		"organizationName": "Org Beta",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(regBodyB))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to register User B: %s", w.Body.String())
	}
	var authB service.AuthResponse
	_ = json.Unmarshal(w.Body.Bytes(), &authB)

	// 3. User A creates Project Alpha
	projBodyA, _ := json.Marshal(map[string]string{"name": "Project Alpha"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/projects", bytes.NewBuffer(projBodyA))
	req.Header.Set("Authorization", "Bearer "+authA.AccessToken)
	req.Header.Set("x-organization-id", authA.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create Project Alpha: %s", w.Body.String())
	}
	var projA service.ProjectResponse
	_ = json.Unmarshal(w.Body.Bytes(), &projA)

	// 4. User A creates Endpoint Alpha
	epBodyA, _ := json.Marshal(map[string]string{"name": "Stripe Webhook", "mode": "DEVELOPMENT"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/projects/"+projA.ID+"/endpoints", bytes.NewBuffer(epBodyA))
	req.Header.Set("Authorization", "Bearer "+authA.AccessToken)
	req.Header.Set("x-organization-id", authA.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create Endpoint Alpha: %s", w.Body.String())
	}
	var epA service.EndpointResponse
	_ = json.Unmarshal(w.Body.Bytes(), &epA)

	// 5. User B attempts to access Project Alpha -> MUST BE 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projects/"+projA.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authB.AccessToken)
	req.Header.Set("x-organization-id", authB.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when User B accesses Project Alpha, got HTTP %d: %s", w.Code, w.Body.String())
	}

	// 6. User B attempts to list endpoints of Project Alpha -> MUST BE 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projects/"+projA.ID+"/endpoints", nil)
	req.Header.Set("Authorization", "Bearer "+authB.AccessToken)
	req.Header.Set("x-organization-id", authB.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when User B lists endpoints of Project Alpha, got HTTP %d", w.Code)
	}

	// 7. User B attempts to access mocks of Endpoint Alpha -> MUST BE 403 Forbidden
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/endpoints/"+epA.ID+"/mocks", nil)
	req.Header.Set("Authorization", "Bearer "+authB.AccessToken)
	req.Header.Set("x-organization-id", authB.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden when User B accesses mocks of Endpoint Alpha, got HTTP %d", w.Code)
	}

	// 8. User A accesses their own Project Alpha -> MUST BE 200 OK
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/projects/"+projA.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authA.AccessToken)
	req.Header.Set("x-organization-id", authA.Organization.ID)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK when User A accesses Project Alpha, got HTTP %d", w.Code)
	}
}
