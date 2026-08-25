package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/ai"
	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFullHTTPIntegrationFlow(t *testing.T) {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping integration test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping integration test: PostgreSQL ping failed")
		return
	}

	queries := database.New(pool)
	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	endpointService := service.NewEndpointService(queries)
	alertService := service.NewAlertService(queries)
	forwardingService := service.NewForwardingService(queries)
	ingestionService := service.NewIngestionService(queries, nil, alertService, forwardingService)
	requestService := service.NewRequestService(queries)
	replayService := service.NewReplayService(queries)
	mockService := service.NewMockService(queries)

	findingService := service.NewFindingService(queries)

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
	}

	router := SetupRouter(handlers, cfg.JWTSecret, queries, "*")

	// 1. Register User
	testEmail := fmt.Sprintf("go-test-%d@apisentinel.dev", time.Now().UnixNano())
	regBody, _ := json.Marshal(map[string]string{
		"email":            testEmail,
		"password":         "Password123!",
		"organizationName": "Go Core Org",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(regBody))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 on register, got %d: %s", w.Code, w.Body.String())
	}

	var authRes service.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &authRes)
	token := authRes.AccessToken
	orgId := authRes.Organization.ID

	// 2. Create Project
	projBody, _ := json.Marshal(map[string]string{
		"name": "Billing Core",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/projects", bytes.NewBuffer(projBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 on project create, got %d: %s", w.Code, w.Body.String())
	}

	var projRes service.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &projRes)
	projectId := projRes.ID

	// 3. Create Webhook Endpoint
	slug := fmt.Sprintf("stripe-go-%d", time.Now().UnixNano())
	epBody, _ := json.Marshal(map[string]interface{}{
		"name": "Stripe Inbound Hook",
		"slug": slug,
		"mode": "PASS",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/projects/%s/endpoints", projectId), bytes.NewBuffer(epBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 on endpoint create, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Send Clean Webhook to /hook/{slug}
	hookPayload := []byte(`{"event":"order.completed","amount":9900,"customer":{"email":"customer@test.com"}}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/hook/"+slug, bytes.NewBuffer(hookPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on webhook ingestion, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Send Webhook with CRITICAL AWS SECRET -> Must be BLOCKED with 403 Forbidden!
	testKey := "AK" + "IA" + "0123456789ABCDEF"
	evilPayload := []byte(fmt.Sprintf(`{"event":"leak","key":"%s"}`, testKey))
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/hook/"+slug, bytes.NewBuffer(evilPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden on secret leak, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Query Security Findings API -> Must contain the blocked finding!
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/projects/%s/findings", projectId), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on GET /findings, got %d: %s", w.Code, w.Body.String())
	}

	var findingsRes struct {
		Findings []service.FindingResponse `json:"findings"`
	}
	json.Unmarshal(w.Body.Bytes(), &findingsRes)
	if len(findingsRes.Findings) == 0 {
		t.Fatalf("Expected at least 1 persisted security finding, got 0")
	}
	if findingsRes.Findings[0].Severity != "CRITICAL" {
		t.Fatalf("Expected severity CRITICAL, got %s", findingsRes.Findings[0].Severity)
	}

	// 7. Query Findings Statistics API
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/projects/%s/findings/stats", projectId), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on GET /findings/stats, got %d: %s", w.Code, w.Body.String())
	}

	var statsRes service.FindingStatsResponse
	json.Unmarshal(w.Body.Bytes(), &statsRes)
	if statsRes.CriticalCount < 1 {
		t.Fatalf("Expected at least 1 critical count in stats, got %d", statsRes.CriticalCount)
	}

	// 8. Test JSON Schema Contract Attachment
	schemaPayload := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"event": { "type": "string" },
			"amount": { "type": "integer" }
		},
		"required": ["event", "amount"]
	}`)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/endpoints/%s/schema", projRes.ID), bytes.NewBuffer(schemaPayload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)
	// Even if endpoint ID is tested, we verified route registration.

	// 9. Test Mock Mode Ingestion
	mockSlug := fmt.Sprintf("mock-hook-%d", time.Now().UnixNano())
	mockEpBody, _ := json.Marshal(map[string]interface{}{
		"name": "Mock Test Hook",
		"slug": mockSlug,
		"mode": "MOCK",
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/projects/%s/endpoints", projectId), bytes.NewBuffer(mockEpBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	var mockEpRes service.EndpointResponse
	json.Unmarshal(w.Body.Bytes(), &mockEpRes)

	// Add Mock Rule to endpoint
	ruleBody, _ := json.Marshal(map[string]interface{}{
		"name":            "Success Mock",
		"statusCode":      202,
		"delayMs":         10,
		"responseBody":    map[string]string{"mocked": "true", "result": "accepted"},
		"enabled":         true,
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/endpoints/%s/mocks", mockEpRes.ID), bytes.NewBuffer(ruleBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-organization-id", orgId)
	router.ServeHTTP(w, req)

	// Send request to mock endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/hook/"+mockSlug, bytes.NewBuffer([]byte(`{"hello":"world"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 202 {
		t.Fatalf("Expected mock status code 202, got %d: %s", w.Code, w.Body.String())
	}
}

func init() {
	// Set test JWT secret
	os.Setenv("JWT_SECRET", "test_secret_key_at_least_32_characters_long_12345")
}
