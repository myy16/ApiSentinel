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

	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRBAC_RoleBasedAccessControl(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping RBAC test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping RBAC test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	endpointService := service.NewEndpointService(queries)
	alertService := service.NewAlertService(queries, nil, cfg.JWTSecret)
	forwardingService := service.NewForwardingService(queries, nil, cfg.JWTSecret)
	requestService := service.NewRequestService(queries)
	findingService := service.NewFindingService(queries)
	apiKeyService := service.NewAPIKeyService(queries)
	webhookSecurityService := service.NewWebhookSecurityService(queries, cfg.JWTSecret)

	handlers := &Handlers{
		AuthHandler:            NewAuthHandler(authService),
		ProjectHandler:         NewProjectHandler(projectService),
		EndpointHandler:        NewEndpointHandler(endpointService),
		AlertHandler:           NewAlertHandler(alertService),
		ForwardingHandler:      NewForwardingHandler(forwardingService),
		RequestHandler:         NewRequestHandler(requestService),
		FindingHandler:         NewFindingHandler(findingService),
		APIKeyHandler:          NewAPIKeyHandler(apiKeyService),
		WebhookSecurityHandler: NewWebhookSecurityHandler(webhookSecurityService),
	}

	router := SetupRouter(handlers, cfg.JWTSecret, queries, "")

	// 1. Setup OWNER User
	ownerEmail := fmt.Sprintf("owner_%d@apisentinel.dev", time.Now().UnixNano())
	ownerAuth, err := authService.Register(ctx, ownerEmail, "Password123!", "RBAC Organization")
	if err != nil {
		t.Fatalf("Failed to register owner: %v", err)
	}
	orgID := ownerAuth.Organization.ID
	ownerToken := ownerAuth.AccessToken

	// 2. Setup DEVELOPER User in the same Org
	devEmail := fmt.Sprintf("dev_%d@apisentinel.dev", time.Now().UnixNano())
	devAuth, err := authService.Register(ctx, devEmail, "Password123!", "Temp Dev Org")
	if err != nil {
		t.Fatalf("Failed to register dev: %v", err)
	}
	devToken := devAuth.AccessToken
	devUUID, _ := uuid.Parse(devAuth.User.ID)
	orgUUID, _ := uuid.Parse(orgID)

	// Add dev to the main org with DEVELOPER role
	_, err = pool.Exec(ctx, "INSERT INTO memberships (organization_id, user_id, role) VALUES ($1, $2, 'DEVELOPER')",
		pgtype.UUID{Bytes: orgUUID, Valid: true},
		pgtype.UUID{Bytes: devUUID, Valid: true},
	)
	if err != nil {
		t.Fatalf("Failed to add dev membership: %v", err)
	}

	// 3. Setup VIEWER User in the same Org
	viewerEmail := fmt.Sprintf("viewer_%d@apisentinel.dev", time.Now().UnixNano())
	viewerAuth, err := authService.Register(ctx, viewerEmail, "Password123!", "Temp Viewer Org")
	if err != nil {
		t.Fatalf("Failed to register viewer: %v", err)
	}
	viewerToken := viewerAuth.AccessToken
	viewerUUID, _ := uuid.Parse(viewerAuth.User.ID)

	// Add viewer to the main org with VIEWER role
	_, err = pool.Exec(ctx, "INSERT INTO memberships (organization_id, user_id, role) VALUES ($1, $2, 'VIEWER')",
		pgtype.UUID{Bytes: orgUUID, Valid: true},
		pgtype.UUID{Bytes: viewerUUID, Valid: true},
	)
	if err != nil {
		t.Fatalf("Failed to add viewer membership: %v", err)
	}

	// 4. Owner creates Project
	proj, err := projectService.CreateProject(ctx, orgID, "RBAC Security Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// --- SCENARIO A: VIEWER Permissions ---
	// VIEWER can GET /api/projects
	reqList, _ := http.NewRequest("GET", "/api/projects", nil)
	reqList.Header.Set("Authorization", "Bearer "+viewerToken)
	reqList.Header.Set("x-organization-id", orgID)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Errorf("Expected VIEWER to list projects with 200 OK, got: %d", wList.Code)
	}

	// VIEWER CANNOT create endpoint (POST /api/projects/{projectId}/endpoints) -> 403 Forbidden
	epBody, _ := json.Marshal(map[string]interface{}{
		"name": "Viewer Unauthorized Hook",
		"slug": fmt.Sprintf("unauth-%d", time.Now().UnixNano()),
		"mode": "DEVELOPMENT",
	})
	reqCreateEpViewer, _ := http.NewRequest("POST", fmt.Sprintf("/api/projects/%s/endpoints", proj.ID), bytes.NewBuffer(epBody))
	reqCreateEpViewer.Header.Set("Authorization", "Bearer "+viewerToken)
	reqCreateEpViewer.Header.Set("x-organization-id", orgID)
	reqCreateEpViewer.Header.Set("Content-Type", "application/json")
	wCreateEpViewer := httptest.NewRecorder()
	router.ServeHTTP(wCreateEpViewer, reqCreateEpViewer)
	if wCreateEpViewer.Code != http.StatusForbidden {
		t.Errorf("Expected VIEWER creating endpoint to return 403 Forbidden, got: %d", wCreateEpViewer.Code)
	}

	// --- SCENARIO B: DEVELOPER Permissions ---
	// DEVELOPER CAN create endpoint (POST /api/projects/{projectId}/endpoints) -> 200/201 OK
	epDevBody, _ := json.Marshal(map[string]interface{}{
		"name": "Dev Authorized Hook",
		"slug": fmt.Sprintf("auth-dev-%d", time.Now().UnixNano()),
		"mode": "DEVELOPMENT",
	})
	reqCreateEpDev, _ := http.NewRequest("POST", fmt.Sprintf("/api/projects/%s/endpoints", proj.ID), bytes.NewBuffer(epDevBody))
	reqCreateEpDev.Header.Set("Authorization", "Bearer "+devToken)
	reqCreateEpDev.Header.Set("x-organization-id", orgID)
	reqCreateEpDev.Header.Set("Content-Type", "application/json")
	wCreateEpDev := httptest.NewRecorder()
	router.ServeHTTP(wCreateEpDev, reqCreateEpDev)
	if wCreateEpDev.Code != http.StatusOK && wCreateEpDev.Code != http.StatusCreated {
		t.Errorf("Expected DEVELOPER creating endpoint to return 200/201, got: %d", wCreateEpDev.Code)
	}

	var createdEp struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(wCreateEpDev.Body.Bytes(), &createdEp)

	// DEVELOPER CANNOT delete endpoint (DELETE /api/projects/{projectId}/endpoints/{endpointId}) -> 403 Forbidden
	reqDelEpDev, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/projects/%s/endpoints/%s", proj.ID, createdEp.ID), nil)
	reqDelEpDev.Header.Set("Authorization", "Bearer "+devToken)
	reqDelEpDev.Header.Set("x-organization-id", orgID)
	wDelEpDev := httptest.NewRecorder()
	router.ServeHTTP(wDelEpDev, reqDelEpDev)
	if wDelEpDev.Code != http.StatusForbidden {
		t.Errorf("Expected DEVELOPER deleting endpoint to return 403 Forbidden, got: %d", wDelEpDev.Code)
	}

	// DEVELOPER CANNOT delete project (DELETE /api/projects/{id}) -> 403 Forbidden
	reqDelProjDev, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/projects/%s", proj.ID), nil)
	reqDelProjDev.Header.Set("Authorization", "Bearer "+devToken)
	reqDelProjDev.Header.Set("x-organization-id", orgID)
	wDelProjDev := httptest.NewRecorder()
	router.ServeHTTP(wDelProjDev, reqDelProjDev)
	if wDelProjDev.Code != http.StatusForbidden {
		t.Errorf("Expected DEVELOPER deleting project to return 403 Forbidden, got: %d", wDelProjDev.Code)
	}

	// --- SCENARIO C: OWNER Permissions ---
	// OWNER CAN delete endpoint (DELETE /api/projects/{projectId}/endpoints/{endpointId}) -> 200 OK
	reqDelEpOwner, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/projects/%s/endpoints/%s", proj.ID, createdEp.ID), nil)
	reqDelEpOwner.Header.Set("Authorization", "Bearer "+ownerToken)
	reqDelEpOwner.Header.Set("x-organization-id", orgID)
	wDelEpOwner := httptest.NewRecorder()
	router.ServeHTTP(wDelEpOwner, reqDelEpOwner)
	if wDelEpOwner.Code != http.StatusOK {
		t.Errorf("Expected OWNER deleting endpoint to return 200 OK, got: %d", wDelEpOwner.Code)
	}

	// OWNER CAN delete project (DELETE /api/projects/{id}) -> 200 OK
	reqDelProjOwner, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/projects/%s", proj.ID), nil)
	reqDelProjOwner.Header.Set("Authorization", "Bearer "+ownerToken)
	reqDelProjOwner.Header.Set("x-organization-id", orgID)
	wDelProjOwner := httptest.NewRecorder()
	router.ServeHTTP(wDelProjOwner, reqDelProjOwner)
	if wDelProjOwner.Code != http.StatusOK {
		t.Errorf("Expected OWNER deleting project to return 200 OK, got: %d", wDelProjOwner.Code)
	}
}
